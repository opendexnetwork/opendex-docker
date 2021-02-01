from urllib.request import urlopen
import subprocess


def checkout(version):
    if version == "latest":
        ref = "master"
    else:
        ref = "v" + version

    url = f"https://github.com/BoltzExchange/boltz-lnd/archive/{ref}.tar.gz"
    with open("src.tar.gz", "wb") as f:
        print("Downloading %s" % url)
        f.write(urlopen(url).read())

    output = subprocess.check_output(f"git ls-remote https://github.com/BoltzExchange/boltz-lnd {ref}", shell=True)
    revision, full_ref = output.decode().strip().split()

    return {
        "GIT_REVISION": revision[:7]
    }
