import os
import platform
import re
import subprocess
import sys
from concurrent.futures import ThreadPoolExecutor, wait, FIRST_EXCEPTION
from subprocess import check_output
from typing import List
import functools
import importlib
from multiprocessing import Process


class Toolkit:
    def __init__(self, project_dir, platforms, group, label_prefix, project_repo):
        self.project_dir = project_dir
        self.group = group
        self.label_prefix = label_prefix
        self.platforms = platforms
        self.project_repo = project_repo

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
        return ["bitcoind"]

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

        m = importlib.import_module(f"{name}.src")

        build_args = []
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

        if platform == self.current_platform:
            cmd = ["docker", "build", ".", "-t", f"{name}:{tag}", "--progress", "plain"]
        else:
            cmd = ["docker", "buildx", "build", ".", "-t", f"{name}:{tag}", "--progress", "plain", "--platform",
                   platform, "--load"]

        cmd.extend(build_args)

        # capture_output=True, text=True
        r = subprocess.run(cmd, shell=False, cwd=image_dir, stdout=sys.stdout, stderr=sys.stderr)
        if r.returncode != 0:
            sys.exit(r.returncode)

        if push:
            print("push images...")


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
