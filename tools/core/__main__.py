import argparse
import os
import json
import subprocess
import sys
from . import Toolkit

parser = argparse.ArgumentParser()
subparsers = parser.add_subparsers(dest="command")

build_parser = subparsers.add_parser("build", prog="build")
build_parser.add_argument("--platform", "-p", action="append")
build_parser.add_argument("--push", action="store_true")
build_parser.add_argument("--modified-images", action="store_true")
build_parser.add_argument("images", type=str, nargs="*")

args = parser.parse_args()

project_dir = os.path.dirname(os.path.dirname(os.path.dirname(__file__)))

t = Toolkit(
    project_dir=project_dir,
    platforms=["linux/amd64", "linux/arm64"],
    group="opendexnetwork",
    label_prefix="network.opendex",
    project_repo="https://github.com/opendexnetwork/opendex-docker",
)

try:
    if args.command == "build":
        if args.modified_images:
            print(json.dumps(t.modified_images))
        else:
            t.build(platforms=args.platform, images=args.images, push=args.push)
    elif args.command == "test":
        t.test()
    elif args.command == "release":
        t.release()
except KeyboardInterrupt:
    pass
except subprocess.CalledProcessError:
    sys.exit(1)
