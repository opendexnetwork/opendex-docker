import functools
import importlib
import json
import os
import platform
import re
import subprocess
import sys
from concurrent.futures import ThreadPoolExecutor, wait, FIRST_EXCEPTION
from subprocess import check_output, check_call
from typing import List
from urllib.request import urlopen, Request
from urllib.error import HTTPError


def run(cmd, capture_output=False):
    print("\033[34m%s\033[0m" % cmd)
    if capture_output:
        return check_output(cmd, shell=True)
    else:
        check_call(cmd, shell=True)


class Toolkit:
    def __init__(self, project_dir, platforms, group, label_prefix, project_repo):
        self.project_dir = project_dir
        self.group = group
        self.label_prefix = label_prefix
        self.platforms = platforms
        self.project_repo = project_repo
        self.token_url = "https://auth.docker.io/token"
        self.registry_url = "https://registry-1.docker.io"

    def _get_current_branch(self) -> str:
        cmd = "git branch --show-current"
        print("\033[34m$ %s\033[0m" % cmd, flush=True)
        output = check_output(cmd, shell=True)
        output = output.decode()
        print("%s" % output.rstrip(), flush=True)
        output = output.strip()
        return output

    def _get_modified_images(self) -> List[str]:
        branch = self._get_current_branch()
        if branch == "master":
            cmd = "git diff --name-only HEAD^..HEAD images"
        else:
            cmd = "git diff --name-only $(git merge-base --fork-point origin/master)..HEAD images"
        print("\033[34m$ %s\033[0m" % cmd, flush=True)
        output = check_output(cmd, shell=True)
        output = output.decode()
        print("%s" % output.rstrip(), flush=True)
        lines = output.splitlines()
        p = re.compile(r"^images/(.+?)/.*$")
        images = set()
        for line in lines:
            m = p.match(line)
            assert m, "mismatch line (%r): %s" % (p, line)
            image = m.group(1)
            images.add(image)

        print()
        print("Modified images: " + ", ".join(images))
        print()

        try:
            images.remove("utils")
        except KeyError:
            pass

        return list(images)

    @functools.cached_property
    def current_platform(self):
        m = platform.machine()
        if m == "x86_64" or m == "AMD64":
            return "linux/amd64"
        elif m == "aarch64":
            return "linux/arm64"
        else:
            raise RuntimeError("Unsupported machine type: " + m)

    @functools.cached_property
    def current_branch(self):
        output = check_output("git branch --show-current", shell=True, cwd=self.project_dir)
        return output.decode().strip()

    @property
    def dirty(self):
        try:
            check_output("git diff --quiet", shell=True, cwd=self.project_dir)
            return False
        except subprocess.CalledProcessError:
            return True

    @functools.cached_property
    def modified_images(self):
        if self.dirty:
            cmd = "git diff --name-only images/"
        else:
            if self.current_branch == "master":
                cmd = "git log --name-only --pretty='format:' -1 -p images/"
            else:
                cmd = "git diff --name-only $(git merge-base --fork-point origin/master)..HEAD images/"

        output = check_output(cmd, shell=True, cwd=self.project_dir)
        images = set()
        for line in output.decode().splitlines():
            parts = line.split("/")
            if len(parts) >= 2:
                images.add(parts[1])
        # return list(images)
        return []

    def _build(self, image, platform, push):
        if ":" in image:
            name, tag = image.split(":")
        else:
            name = image
            tag = "latest"

        images_dir = os.path.join(self.project_dir, "images")
        if images_dir not in sys.path:
            sys.path.append(images_dir)

        image_dir = os.path.join(images_dir, name)

        build_args = []

        try:
            m = importlib.import_module(f"{name}.src")
            wd = os.getcwd()
            try:
                os.chdir(image_dir)
                args = m.checkout(tag)
                if args:
                    for key, value in args.items():
                        build_args.extend([
                            "--build-arg", f"{key}={value}"
                        ])
            finally:
                os.chdir(wd)
        except ModuleNotFoundError:
            pass

        if self.current_branch == "master":
            full_tag = f"{self.group}/{name}:{tag}"
        else:
            branch = self.current_branch.replace("/", "-")
            full_tag = f"{self.group}/{name}:{tag}__{branch}"

        dockerfile = "Dockerfile"
        if platform == "linux/arm64":
            dockerfile = "Dockerfile.aarch64"

        if not os.path.exists(os.path.join(image_dir, dockerfile)):
            dockerfile = "Dockerfile"

        build_opts = ["--progress", "plain", "-t", full_tag, "-f", dockerfile]

        build_opts.extend(build_args)

        build_opts.append(".")

        if platform == self.current_platform:
            cmd = ["docker", "build"] + build_opts
        else:
            cmd = ["docker", "buildx", "build", "--platform", platform, "--load"] + build_opts

        print("\033[34m%s\033[0m" % " ".join(cmd))
        subprocess.check_call(cmd, shell=False, cwd=image_dir, stdout=sys.stdout, stderr=sys.stderr)

        if push:
            self.push(full_tag, platform)

    def push(self, tag, platform):
        arch = platform.split("/")[1]  # amd64, arm64
        arch_tag = f"{tag}__{arch}"
        run(f"docker tag {tag} {arch_tag}")

        output = run("docker push {}".format(arch_tag), capture_output=True)
        last_line = output.decode().splitlines()[-1]
        p = re.compile(r"^(.*): digest: (.*) size: (\d+)$")
        m = p.match(last_line)
        digest = m.group(2)
        print(digest)

        print("Updating manifest list", tag)
        os.environ["DOCKER_CLI_EXPERIMENTAL"] = "enabled"

        repo, _ = tag.split(":")
        subprocess.run(f"docker manifest rm {tag}", shell=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

        manifests = self.get_manifests(tag)
        manifests[platform] = digest
        tags = []
        for p, digest in manifests.items():
            tags.append("{}@{}".format(repo, digest))

        run("docker manifest create {} {}".format(tag, " ".join(tags)))
        run("docker manifest push -p {}".format(tag))

    def get_token(self, repo):
        r = urlopen("{}?service=registry.docker.io&scope=repository:{}:pull".format(self.token_url, repo))
        return json.loads(r.read().decode())["token"]

    def get_manifests(self, full_tag):
        result = {}

        print("Inspecting manifest", full_tag)
        try:
            repo, tag = full_tag.split(":")
            url = f"{self.registry_url}/v2/{repo}/manifests/{tag}"
            req = Request(url)
            req.add_header("Authorization", "Bearer " + self.get_token(repo))
            media_types = [
                "application/vnd.docker.distribution.manifest.list.v2+json",
                "application/vnd.docker.distribution.manifest.v2+json",
                "application/vnd.docker.distribution.manifest.v1+json",
            ]
            req.add_header("Accept", ",".join(media_types))
            r = urlopen(req)

            j = json.load(r)
            print(j)
            if "manifests" in j:
                for m in j["manifests"]:
                    p = "{}/{}".format(m["platform"]["os"], m["platform"]["architecture"])
                    result[p] = m["digest"]
        except HTTPError as e:
            if e.code != 404:
                raise e
            else:
                print("Not found")

        return result

    def build(self, platforms, images, push):
        if not platforms or len(platforms) == 0:
            platforms = [self.current_platform]

        if not images or len(images) == 0:
            images = self.modified_images

        futs = []

        with ThreadPoolExecutor(max_workers=1, thread_name_prefix="worker") as executor:
            for image in images:
                for p in platforms:
                    futs.append(executor.submit(self._build, image, p, push))

        finished, pending = wait(futs, timeout=None, return_when=FIRST_EXCEPTION)
        for task in finished:
            task.result()

    def test(self):
        print("to be implemented")

    def release(self):
        print("to be implemented")
