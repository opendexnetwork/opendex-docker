from urllib.request import urlopen
import json
import os
import shutil

REPO = "lightningnetwork/lnd"


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
        ref = "v0.12.1-beta"
    else:
        ref = "v" + version

    revision = ensure_src(version, ref)

    tags = [
        "autopilotrpc",
        "chainrpc",
        "invoicesrpc",
        "routerrpc",
        "signrpc",
        "walletrpc",
        "watchtowerrpc",
        "wtclientrpc",
        "experimental",
    ]
    args = {
        "TAGS": " ".join(tags),
        "LDFLAGS": "-X github.com/lightningnetwork/lnd/build.Commit={}".format(revision),
    }

    return args
