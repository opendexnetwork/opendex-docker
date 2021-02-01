from urllib.request import urlopen
import json
import os
import shutil

REPO = "ethereum/go-ethereum"


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


def checkout(version):
    if version == "latest":
        ref = "v1.9.24"
    else:
        ref = "v" + version

    ensure_src(version, ref)
