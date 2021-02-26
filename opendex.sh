#!/bin/sh

set -e

OPENDEX_LAUNCHER_VERSION="v1.0.0-rc.3"

# ensure the opendex-launcher binary is downloaded
# select the network
# run opendex-launcher setup --interactive

if [ "$(uname)" = "Darwin" ]; then
  OPENDEX_DOCKER_HOME="$HOME/Library/Application\ Support/OpendexDocker"
else
  OPENDEX_DOCKER_HOME="$HOME/.opendex-docker"
fi

if ! [ -e "$OPENDEX_DOCKER_HOME" ]; then
  mkdir "$OPENDEX_DOCKER_HOME"
fi

ensure_launcher() {
  LAUNCHER=${LAUNCHER:-"/usr/bin/opendex-launcher"}
}

ensure_network() {
  while [ -z "${NETWORK:-}" ]; do
    echo "1) Testnet"
    echo "2) Mainnet"
    read -r -p "Please choose the network: "
    case $REPLY in
      1) NETWORK="testnet";;
      2) NETWORK="mainnet";;
    esac
  done
}

ensure_launcher
ensure_network
$LAUNCHER setup --interactive
