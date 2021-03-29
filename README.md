# opendex-docker

[![Discord](https://img.shields.io/discord/628640072748761118.svg)](https://discord.gg/aS5RMchDrU)
[![Go Report Card](https://goreportcard.com/badge/github.com/opendexnetwork/opendex-docker)](https://goreportcard.com/report/github.com/opendexnetwork/opendex-docker)
[![Build](https://github.com/opendexnetwork/opendex-docker/workflows/Build/badge.svg)](https://github.com/opendexnetwork/opendex-docker/actions?query=workflow%3ABuild)
[![Docker Pulls](https://img.shields.io/docker/pulls/opendexnetwork/opendexd)](https://hub.docker.com/r/opendexnetwork/opendexd)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)


A complete [opendexd](https://github.com/opendexnetwork/opendexd) environment using [docker](https://www.docker.com/), made for liquidity providers on the OpenDEX Network.

Two ways to become a liquidity provider:
- run the 👉 [desktop app](https://opendex-desktop.app/) 👈 on your pc or
- use the 👉 [cli-based setup](https://opendex.network/docs) 👈 on your server 

### Basic Usage via CLI

Use command-line user interface by running `opendex.sh` script.

```
bash opendex.sh
```

Use the launcher for advanced control over the whole environment.

```
make launcher
cd launcher && ./launcher help
```
