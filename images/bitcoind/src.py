from urllib.request import urlopen
import subprocess
import os
import shutil


def checkout(version):
    if version == "latest":
        ref = "v0.21.0"
    else:
        ref = "v" + version

    print("Inspecting reference", ref)
    output = subprocess.check_output(f"git ls-remote https://github.com/bitcoin/bitcoin {ref}", shell=True)
    print(output.decode())
    revision, full_ref = output.decode().strip().split()

    if not os.path.exists(".cache"):
        os.mkdir(".cache")

    gz = f".cache/{revision}.tar.gz"
    if not os.path.exists(gz):
        url = f"https://github.com/bitcoin/bitcoin/archive/{revision}.tar.gz"
        with open(gz, "wb") as f:
            print("Downloading %s" % url)
            f.write(urlopen(url).read())

    print("Extracting", gz)
    if os.path.exists(".src"):
        shutil.rmtree(".src")
    else:
        os.mkdir(".src")
    os.system(f"tar xzf {gz} --strip-components 1 -C .src")
