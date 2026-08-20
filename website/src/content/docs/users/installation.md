---
title: Installation
description: Prerequisites and the two ways to install cais — go install or build from source.
sidebar:
  order: 2
---

# Installation

## Prerequisites

- **Docker with the Compose plugin** on your `PATH`. cais drives the `docker compose` CLI, so both must be installed and the daemon running.
- **Go 1.26+** — only needed to build from source; `go install` fetches a prebuilt binary.
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

## A note on binaries

There are no downloadable release binaries yet. A `v*` tag builds them (Linux and macOS, amd64 and arm64), but the releases stay in draft until the launch work in the roadmap — so `go install` or a clone is the way in for now.