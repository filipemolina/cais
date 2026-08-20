---
title: Development Workflow
description: The build/test loop, code style, CI, and the release process.
---

# Development Workflow

## Setup

- **Go 1.26+** to build.
- **Docker with the Compose plugin** on your `PATH` to run the app against a real stack.

## The loop

```bash
make dev           # run against the compose file in the current directory
go build ./...
go vet ./...
go test ./...
gofmt -l .         # must print nothing
```

`make dev` runs `go run main.go`. `make build` builds and installs to `$(go env GOPATH)/bin` (usually `~/go/bin`), stamping the version from `git describe` via `-ldflags`.

CI runs exactly these on every pull request, with `go test -race`. Run the race detector locally too if you touch anything that shells out or streams: the docker calls and the log stream each run on their own goroutine.

**Keep every commit green, not just the branch tip.** A commit that does not build is a commit nobody can bisect through.

## Code style

Match the code around you. Two things that are not obvious from a diff:

- **Comments say why, not what.** The code already says what. A comment earns its place by recording a decision, a constraint, or a trap — ideally the thing that would otherwise be "fixed" back into a bug six months later.
- **A keybinding is declared once**, in `src/keys/Keys.go`. The panels match against it and the footer and `?` overlay render from it, so a key added anywhere else will not be advertised and may collide.

Commit messages: a short summary line, then prose explaining why. Look at `git log` for the register.

## CI

`.github/workflows/ci.yml` runs on every pull request: build, vet, and `go test -race`. `.github/workflows/docs.yml` builds the documentation site on push to main and on PRs.

## Releases

Maintainer-only, and automatic: pushing a `v*` tag runs GoReleaser, which builds for linux and darwin (amd64 and arm64), stamps the version into the binary, and opens a draft release to be reviewed before publishing.

```bash
git tag -a v0.1.0 -m "v0.1.0"
git push origin v0.1.0
```

`cais --version` reports the stamp. An unstamped local build reports its commit instead, which is what a bug report wants anyway.

## Saying which build this is

`constants.Version()` is the single reader of a version that may or may not have been stamped, so no caller has to care whether it was. `make build` and the GoReleaser build both pass `-ldflags -X …constants.version=`, and that value wins when it is there.

When it is not, the build info answers, and which half of it answers separates the two remaining cases without a heuristic:

- A binary built from a checkout has a `vcs.revision` — the short commit is its version, and it is the thing a bug report actually needs.
- A binary from `go install …@v0.1.0` has no VCS information at all, because it was built from a module download, so `Main.Version` is both the only answer and the right one.

The toolchain's synthesized `v0.0.0-<date>-<hash>` pseudo-version is deliberately ignored: it says exactly what the commit says, three times longer, and looks like a release nobody ever made.