from urllib.request import urlopen
import json
import os
import shutil


REPO = "opendexnetwork/opendexd"


def ensure_src(version, ref):
    url = f"https://api.github.com/repos/{REPO}/commits/{ref}"
    r = urlopen(url)
    revision = json.load(r)["sha"]
    print("%s -> %s -> %s" % (version, ref, revision))

    if not os.path.exists(".cache"):
        os.mkdir(".cache")

    gz = f".cache/{revision}.tar.gz"
    if not os.path.exists(gz):
        url = f"https://github.com/{REPO}/archive/{revision}.tar.gz"
        print("Downloading", url)
        with open(gz, "wb") as f:
            f.write(urlopen(url).read())

    print("Extracting", gz)
    if os.path.exists(".src"):
        shutil.rmtree(".src")
    os.mkdir(".src")
    os.system(f"tar xzf {gz} --strip-components 1 -C .src")

    return revision


def checkout(version):
    if version == "latest":
        ref = "main"
    elif version == "1.2.4-1":
        return "v1.2.4"
    else:
        ref = "v" + version

    revision = ensure_src(version, ref)

    return {
        "GIT_REVISION": revision[:8]
    }
