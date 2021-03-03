#!/bin/sh

set -e

LAUNCHER_VERSION="v1.0.0-rc.4"

assemble_launcher_download_url() {
  case $(uname) in
    Darwin) os="darwin";;
    Linux) os="linux";;
    *)
      echo "Unsupported kernel: $(uname)"
      exit 1
      ;;
  esac

  case $(uname -m) in
    x86_64) arch="amd64";;
    aarch64)
      if [ "$os" = "darwin" ]; then
        echo "The arm64 macOS has not been supported yet."
        exit 1
      fi
      arch="arm64"
      ;;
    *)
      echo "Unsupported machine: $(name -m)"
      exit 1
      ;;
  esac

  filename="opendex-launcher-${os}-${arch}.zip"
  unset os
  unset arch

  LAUNCHER_DOWNLOAD_URL="https://github.com/opendexnetwork/opendex-launcher/releases/download/$LAUNCHER_VERSION/$filename"
  unset filename
}

assemble_launcher_download_url

if [ "$(uname)" = "Darwin" ]; then
  OPENDEX_DOCKER_HOME="$HOME/Library/Application\ Support/OpendexDocker"
else
  OPENDEX_DOCKER_HOME="$HOME/.opendex-docker"
fi

if ! [ -e "$OPENDEX_DOCKER_HOME" ]; then
  mkdir "$OPENDEX_DOCKER_HOME"
fi

DEFAULT_LAUNCHER="$OPENDEX_DOCKER_HOME/opendex-launcher"

install_launcher() {
  # Install opendex-launcher binary file into $OPENDEX_DOCKER_HOME folder
  echo "Installing opendex-launcher $LAUNCHER_VERSION ..."
  echo "$LAUNCHER_DOWNLOAD_URL"
  curl -sfL "$LAUNCHER_DOWNLOAD_URL" | tar xf - -C "$OPENDEX_DOCKER_HOME"
  chmod u+x "$DEFAULT_LAUNCHER"
}

ensure_launcher() {
  LAUNCHER=${LAUNCHER:-"$DEFAULT_LAUNCHER"}
  install=false
  if [ -e "$LAUNCHER" ]; then
    if ! "$LAUNCHER" version | head -1 | grep -q "$LAUNCHER_VERSION"; then
      install=true
    fi
  else
    if [ "$LAUNCHER" = "$DEFAULT_LAUNCHER" ]; then
      install=true
    else
      echo "opendex-launcher not found: $LAUNCHER"
      exit 1
    fi
  fi
  $install && install_launcher
  unset install
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
export NETWORK=$NETWORK
"$LAUNCHER" setup --interactive
