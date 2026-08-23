---
title: Installation
description: Prerequisites and the three ways to install cais — a release binary, go install, or build from source.
sidebar:
  order: 2
---

# Installation

## Prerequisites

- **Docker with the Compose plugin** on your `PATH`. cais drives the `docker compose` CLI, so both must be installed and the daemon running.
- **Go 1.26+** — needed for `go install` and for building from source, since both compile cais on your machine. Not needed if you download a release binary.
- **A terminal** — cais is a full-screen TUI.

If something is missing, cais says which one failed (missing Docker, missing Compose plugin, unreachable daemon, or permissions) and gives you the exact command to fix it. It never installs or configures anything on your machine itself.

## Install with `go install`

```bash
go install github.com/filipemolina/cais@latest
```

This installs the `cais` binary into `$(go env GOPATH)/bin` (usually `~/go/bin`). Make sure that directory is on your `PATH`.

## Build from source

```bash
git clone https://github.com/filipemolina/cais.git
cd cais
make build     # installs to $(go env GOPATH)/bin, usually ~/go/bin
```

`make build` stamps the version into the binary from `git describe`, so `cais --version` reports the tag or commit you built.

## Verify

```bash
cais --version
```

An unstamped local build reports its commit hash instead of a version — that is expected, and it is exactly what a bug report wants.

## Download a release binary

Pre-built binaries are attached to every published release: Linux and macOS, amd64 and arm64, with a `checksums.txt` beside them.

Grab the archive for your platform from the [latest release](https://github.com/filipemolina/cais/releases/latest), then:

```bash
tar -xzf cais_*_linux_amd64.tar.gz
sudo install -m 755 cais /usr/local/bin/cais
```

No Go toolchain needed for this path.