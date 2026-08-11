# Plan: Releases People Can Actually Download — Linux, macOS, Windows

> **Before you start.** Work on a feature branch of small commits, merged
> `--no-ff`; `go build ./... && go vet ./... && go test ./... && gofmt -l .`
> green at **every** commit, not just at the tip — `docs/ROADMAP.md`
> §Conventions is the full contract and `CONTRIBUTING.md` explains how a TUI
> gets tested. Behaviour that only shows on screen gets checked in the real app
> with VHS before it is committed. **Step 7 of the post-alpha order.** One exception, worth doing now rather than waiting: cut a `v0.1.0` tag early — several directories require a first release four months old, and the clock starts at the tag.

Ask: *"How to produce releases so people can download them for Linux, Mac and
Windows. It doesn't need to be a release for every single available
architecture and OS, but mainly the ones more used by the self-hosting
community."*

Measured against the repo on 2026-07-31.

## Status — most of this already exists

The release pipeline is built and working; it is the *reach* that is missing.

- **`.goreleaser.yaml`** (v2 schema): one build, `CGO_ENABLED=0`, `goos: [linux,
  darwin]` × `goarch: [amd64, arm64]`, `tar.gz` archives with README + LICENSE,
  `checksums.txt`, a filtered changelog, and `release.draft: true` so a human
  reads the notes before publishing. The version is stamped into
  `src/constants.version` with the same ldflags the Makefile uses, so a
  released binary and a `make build` answer `--version` identically.
- **`.github/workflows/release.yml`**: a `v*` tag triggers
  `goreleaser-action@v6` with `--clean` on `ubuntu-latest`, `fetch-depth: 0`
  for the changelog, `contents: write`, and the default `GITHUB_TOKEN`.
- **`CONTRIBUTING.md` §Releases** documents the `git tag -a` / `git push`
  flow and says the builds are "linux and darwin (amd64 and arm64)".
- **`go install github.com/filipemolina/cais@latest` already works** —
  the module path is public and `main` is at the repo root. That is a real
  install method the README does not mention.

And one recorded decision, in the config itself:

```yaml
# No Windows build: the app shells out to `docker compose` and hands the
# terminal to $EDITOR, neither of which has been tried there.
```

**That comment is a risk register, not a law** — the exact framing
`docs/plans/cross-platform-testing.md` already uses. This plan **depends on**
that one and takes over its Step 3: run the suite on Windows and macOS runners
first (its Phases 1–2), then ship Windows binaries here. Do not implement its
Step 3 and this plan's Phase 1 separately; they are the same work, and it lives
here.

**Verdict up front: worth doing, in four phases of decreasing return.** Phase 1
(the build matrix) is a config change that triples the reachable audience.
Phase 2 (Linux packages + an install line) is what the self-hosting audience
actually uses. Phases 3–4 (Homebrew, Scoop, winget, AUR) each need a *new
repository and a token*, so they are where the free-and-automatic story stops
being free of admin.

## Who the artifacts are for

"The self-hosting community" is a concrete set of machines, and it decides the
matrix:

| Target | Who | Verdict |
|---|---|---|
| `linux/amd64` | every x86 home server, NUC, old desktop, Proxmox VM, Synology/QNAP with Docker, VPS | **essential** |
| `linux/arm64` | Raspberry Pi 4/5 on 64-bit Pi OS, Oracle/Hetzner ARM VPS, Apple-silicon Linux VMs | **essential** — this is where the self-hosting growth is |
| `linux/arm` (v7) | Pi 2/3 and any box still on 32-bit Raspberry Pi OS | **include** — one more archive, `CGO_ENABLED=0` cross-compiles it for free, and this crowd is exactly the "reuse the old Pi" crowd |
| `darwin/arm64` | every Mac since 2020 | **essential** |
| `darwin/amd64` | Intel Macs, still numerous | **include**; drop when it stops being worth the tarball |
| `windows/amd64` | Docker Desktop on Windows, WSL2 users who run the app on the Windows side | **include** — it is the literal ask |
| `windows/arm64` | Snapdragon laptops | **skip for now**; add on the first request. It has no CI coverage and no known user |
| `linux/riscv64`, `freebsd/*`, `linux/386` | vanishingly few | **skip**. FreeBSD self-hosters run jails, not Docker |

Seven artifacts. GoReleaser builds them in one job on `ubuntu-latest` because
cgo is already off.

## Design decisions

### D1. Windows ships only after Windows CI is green

`docs/plans/cross-platform-testing.md` Phases 1–2 add `windows-latest` and
`macos-latest` to the CI matrix (with the `shell: bash` fix its gofmt step
needs) and a boot smoke test. Ship binaries after those are required checks and
passing — not because the code is likely broken, but because the goreleaser
comment names two untested paths (`docker compose` shell-out, `$EDITOR`
handover) and shipping a binary is a promise that they work.

Two Windows facts worth writing into the docs at the same time:

- **Docker Desktop for Windows bundles Compose v2**, so `exec.Command("docker",
  "compose", …)` resolves `docker.exe` from `PATH` and behaves as on Linux.
  Windows Server runners ship Docker + Compose too, which is what makes the
  smoke test possible.
- **`$EDITOR` is usually unset on Windows, and the fallback is `vi`.**
  Confirmed: `utils.FallbackEditor = "vi"` (`src/utils/Editor.go:11`), chosen
  because POSIX requires it — which is exactly why it is absent on Windows.
  `E` and `ctrl+o` will fail there with an exec error until the fallback is
  `notepad` on `GOOS == "windows"`. That is a three-line fix, it is a *code*
  change rather than a packaging one, and it belongs in the cross-platform
  plan's follow-up — but it must land **before** Phase 1 ships a Windows
  binary, because it is a guaranteed first-day bug report.

### D2. Windows archives are `zip`

`tar.gz` is not openable by double-click on Windows. GoReleaser v2 does this
with a per-OS override, in the same `formats` vocabulary the file already uses:

```yaml
archives:
  - formats: [tar.gz]
    format_overrides:
      - goos: windows
        formats: [zip]
    name_template: >-
      {{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}
    files:
      - README.md
      - LICENSE
```

### D3. Linux packages, because "download a tarball" is not how servers work

A self-hoster on Debian wants `apt install ./cais_0.2.0_amd64.deb`, not a
tarball they have to put on `PATH` themselves. GoReleaser's `nfpms` produces
`deb`, `rpm`, `apk`, `archlinux`, `ipk` and more from the binaries already
built. Ship **deb, rpm and apk** — Debian/Ubuntu, Fedora/RHEL, Alpine — which
covers effectively every Docker host.

```yaml
nfpms:
  - id: packages
    package_name: cais
    vendor: Filipe Molina
    maintainer: Filipe Molina <ciriloboy@gmail.com>
    description: A terminal UI for managing Docker Compose stacks.
    license: MIT          # matches LICENSE (MIT, © 2026 Filipe Molina)
    homepage: https://github.com/filipemolina/cais
    formats: [deb, rpm, apk]
```

They are unsigned and not in any distro repository — the user downloads the
file and installs it. That is normal for a young tool and worth one honest
sentence in the README rather than a pretence of packaging maturity.

### D4. An install script, written to be readable

```console
curl -sSfL https://raw.githubusercontent.com/filipemolina/cais/main/scripts/install.sh | sh
```

`curl | sh` deserves the suspicion it gets, so: the README shows the **download
and verify** path first (release URL, `checksums.txt`, `sha256sum -c`), and the
script second, with a line telling the reader to open it first. The script
detects OS and arch, resolves the latest release through the GitHub API,
downloads the archive **and** `checksums.txt`, verifies, and installs to
`/usr/local/bin` (or `$PREFIX`). ~60 lines of POSIX `sh`. No `sudo` inside the
script — if the destination is not writable, print the `sudo` command and exit.

### D5. Provenance for free: GitHub artifact attestations

`checksums.txt` proves the file was not corrupted; it does not prove who built
it. `actions/attest-build-provenance` signs a provenance statement for the
release artifacts with the workflow's own identity — no key to manage, no
account, no cost — and users verify with
`gh attestation verify <file> --repo filipemolina/cais`.

Cosign keyless signing is the alternative; it needs a `cosign` step and gives
roughly the same guarantee with more moving parts. Take the attestation.

### D6. Homebrew: a tap, a cask, and an honest word about Gatekeeper

- **A tap, not homebrew-core.** homebrew-core has notability requirements a
  young project does not meet. A `filipemolina/homebrew-tap` repository gives
  `brew install filipemolina/tap/cais` on day one.
- **`homebrew_casks`, not `brews`.** GoReleaser's `brews` (formula) section is
  deprecated; casks are the current path for distributing a prebuilt binary.
- **The token is the catch.** GoReleaser cannot push to another repository with
  the workflow's default `GITHUB_TOKEN`; the tap needs a PAT (or a GitHub App
  token) with contents-write on the tap repo, stored as a secret.
- **Gatekeeper.** The binaries are unsigned — signing and notarizing needs a
  paid Apple Developer account, which is out of scope for a free release
  pipeline. So a downloaded tarball gets quarantined and macOS refuses to run
  it until the user clears the attribute:

  ```console
  xattr -dr com.apple.quarantine /usr/local/bin/cais
  ```

  GoReleaser's cask can run that as a post-install hook
  (`hooks.post.install` with `system_command "/usr/bin/xattr"`), and its own
  documentation warns plainly that this *"bypasses macOS security protections
  designed to verify software authenticity"* and that Apple may close it. Do
  both: use the hook **and** put the manual command in the cask's `caveats` and
  in the README, so a user who downloads the tarball directly is not stuck.

### D7. Windows package managers: Scoop now, winget when someone asks

- **Scoop** is what Windows CLI users already use, and GoReleaser publishes to
  a bucket repository — one more repo, same PAT. Cheap.
- **winget** requires a fork of `microsoft/winget-pkgs` and a pull request per
  release, reviewed by Microsoft's pipeline. GoReleaser automates opening the
  PR (`winget:` with `pull_request.enabled`, a publisher name, a short
  description and a license), and notably **does not fail the release if the PR
  fails** — it logs and moves on. Worth doing, worth doing *last*, because the
  first submission usually needs a human round trip.
- **Chocolatey** — skip. Overlaps Scoop for this audience.

### D8. AUR: only with a maintainer's appetite

`yay -S cais` is what an Arch self-hoster expects, and GoReleaser publishes a
`aurs` (binary PKGBUILD) to an AUR repository over SSH — which means an AUR
account and a **private SSH key in GitHub secrets**. That is the highest-admin,
lowest-reach item on the list. Do it only if an Arch user asks.

### D9. No Docker image of the TUI (for now)

lazydocker ships one and it is a reasonable thing to want. For this app it
means the image must contain the `docker` CLI *and* the compose plugin, the
container needs `/var/run/docker.sock` mounted plus the user's compose
directory bind-mounted at the same path (or every path in the compose file
breaks), and `$EDITOR` inside the container is not the user's editor. The
result would be a worse version of the app that is harder to explain. Revisit
if people ask; the answer today is "install the binary, it is 12 MB".

## Phases

Feature branch per phase, `--no-ff` merge, ROADMAP row, per
`docs/ROADMAP.md` §Conventions. Every phase ends with a **verified artifact**,
not just a green config.

### Phase 1 — The matrix: Windows, macOS, and 32-bit ARM

**Prerequisite:** `docs/plans/cross-platform-testing.md` Phases 1–2 merged and
green.

| File | Change |
|---|---|
| `.goreleaser.yaml` | `goos: [linux, darwin, windows]`; `goarch: [amd64, arm64, arm]`; `goarm: ["7"]`; `ignore:` the combinations that make no sense (`windows/arm`, `darwin/arm`); `format_overrides` for the Windows zip (D2) |
| `.goreleaser.yaml` | replace the `# No Windows build` comment with what replaced it: the two risks, now covered by CI, and a pointer to the cross-platform plan |
| `CONTRIBUTING.md` | §Releases: "linux and darwin (amd64 and arm64)" → the real matrix |
| `README.md` | Requirements: Windows needs Docker Desktop with the Compose plugin |

Verify locally before tagging anything:

```bash
goreleaser check
goreleaser release --snapshot --clean
ls -la dist/            # 7 archives + checksums.txt
file dist/*windows*/cais.exe
```

Then tag a `v0.x.y-rc.1` and confirm the draft release carries every artifact.

Acceptance: seven archives (`linux_amd64`, `linux_arm64`, `linux_armv7`,
`darwin_amd64`, `darwin_arm64`, `windows_amd64` as **zip**), `checksums.txt`
covering all of them, `cais.exe --version` printing the tag on a Windows
runner, and no config or workflow change beyond `.goreleaser.yaml` and docs.

### Phase 2 — Packages, install script, provenance

| File | Change |
|---|---|
| `.goreleaser.yaml` | `nfpms:` deb/rpm/apk (D3) |
| `scripts/install.sh` | new (D4), POSIX `sh`, checksum-verifying, no `sudo` |
| `.github/workflows/release.yml` | `actions/attest-build-provenance` after the goreleaser step; `permissions:` gains `id-token: write` and `attestations: write` (D5) |
| `README.md` | an Install section: package downloads, the verify commands, `go install`, the script |

Acceptance: `dist/` contains `.deb`, `.rpm` and `.apk`; each installs in a
throwaway container and `cais --version` works —

```bash
docker run --rm -v "$PWD/dist:/d" debian:12  sh -c 'apt-get install -y /d/cais_*_amd64.deb && cais --version'
docker run --rm -v "$PWD/dist:/d" fedora:41  sh -c 'dnf install -y /d/cais*.x86_64.rpm && cais --version'
docker run --rm -v "$PWD/dist:/d" alpine:3   sh -c 'apk add --allow-untrusted /d/cais_*.x86_64.apk && cais --version'
```

— the install script works on a clean container, and
`gh attestation verify` passes against a published artifact.

### Phase 3 — Homebrew tap and Scoop bucket

**Needs two new repositories and one secret**, which is the only external
dependency in this plan:

1. `filipemolina/homebrew-tap` (public, empty)
2. `filipemolina/scoop-bucket` (public, empty)
3. a PAT with contents-write on both, stored as `HOMEBREW_TAP_TOKEN` (or a
   single `PUBLISH_TOKEN` used by both), added to `release.yml`'s env

| File | Change |
|---|---|
| `.goreleaser.yaml` | `homebrew_casks:` with the tap repo, `hooks.post.install` xattr, and `caveats` (D6); `scoops:` with the bucket repo (D7) |
| `.github/workflows/release.yml` | pass the token in `env` |
| `README.md` | `brew install filipemolina/tap/cais`, `scoop install cais` |

Acceptance: after a tagged release, the tap and bucket repos each contain a
generated manifest for the new version; `brew install` from the tap produces a
working `cais` on macOS; `scoop install` produces a working `cais.exe`.

### Phase 4 — winget, and AUR only on demand

`winget:` in `.goreleaser.yaml` with `pull_request.enabled: true`, a fork of
`microsoft/winget-pkgs` as the repository and the upstream as the PR base
(D7). Expect the first PR to need manual attention; the release does not fail
if it does.

`aurs:` only if an Arch user asks (D8), and only with a dedicated deploy key.

Acceptance: a winget PR opens automatically on tag; once merged,
`winget install cais` works.

## What is decided, and what still needs the owner

Unlike the other plans in this folder, this one has **external dependencies**:

| Item | Needs | Free? |
|---|---|---|
| Phases 1–2 | nothing — repo-local | yes |
| Homebrew tap, Scoop bucket | two new public repos + a PAT secret | yes |
| winget | a fork of `microsoft/winget-pkgs`; Microsoft reviews | yes |
| AUR | an AUR account + an SSH deploy key secret | yes |
| **macOS code signing / notarization** | an Apple Developer account, ~$99/year | **no — out of scope**, hence D6's xattr path |

**The technical choices are decided,** not recommended: include `linux/armv7`,
skip `windows/arm64`, ship deb/rpm/apk, take the attestation over cosign, tap
before core, Scoop before winget, no Docker image, no Apple account. Implement
them as written.

**What genuinely still needs you** — everything below Phase 2 requires
something no implementer can create on their own, so a phase that reaches one
of these stops there and asks:

| Blocked on | What it needs from the owner |
|---|---|
| Homebrew tap, Scoop bucket (Phase 3) | Two new public repos, and a PAT stored as an Actions secret |
| winget (Phase 4) | A fork of `microsoft/winget-pkgs`, and Microsoft's review |
| AUR (Phase 4, on demand only) | An AUR account and an SSH deploy key secret |
| macOS signing/notarization | An Apple Developer account at ~$99/year — **out of scope**, which is why D6 documents the xattr path instead |

Phases 1 and 2 are repo-local and need none of it. Those are the ones to do.

## Edge cases and risks

1. **`linux/armv7` has no CI coverage.** It compiles; nobody runs the tests on
   it. Say so in the release notes rather than implying support. (Adding an ARM
   runner is a separate conversation.)
2. **Windows terminal capability.** Windows Terminal and modern PowerShell
   handle the escape sequences Bubble Tea emits; legacy `conhost` on old
   Windows 10 builds may not. Bubble Tea v2 pulls in
   `charmbracelet/x/windows`, so the framework supports it — but the boot smoke
   test from the cross-platform plan is what turns that from a claim into
   evidence.
3. **`$EDITOR` on Windows** — D1. Find out what `utils.Editor` falls back to
   before the first Windows user does.
4. **Gatekeeper on the tarball path** — D6; the cask hook does not help someone
   who downloaded the `.tar.gz`, so the README must carry the `xattr` line.
5. **A draft release with 15+ artifacts** is a lot to eyeball. Keep
   `release.draft: true` anyway — the changelog is the thing being reviewed,
   not the file list.
6. **Version stamping already works** and must keep working: the ldflags path
   `github.com/filipemolina/cais/src/constants.version` appears in
   both the Makefile and `.goreleaser.yaml`. If the module path ever changes,
   both change together, and `cais --version` is the test.
7. **`checksums.txt` name collisions** across formats — nfpm artifacts are
   included in the checksum file by default; confirm in the snapshot output.
8. **Tag hygiene.** The changelog is built from commits since the last tag and
   needs `fetch-depth: 0`, which `release.yml` already sets. A re-tagged
   release produces a confusing changelog; prefer a new patch tag over moving
   one.
9. **Homebrew cask naming.** A cask named `cais` may collide in a user's tap
   namespace with something else; the tap-qualified install
   (`filipemolina/tap/cais`) is unambiguous and is what the README should
   show.
10. **The binary name is `cais`, the module is `cais`.** Package
    names, cask names and the install script must all use `cais`; only
    the repository and module keep the long name. Sweep for this before
    Phase 2 — an install that puts `cais` on `PATH` while the README,
    docs and cask names all use the same name keeps the project
    coherent forever.

## Do not

- Do not ship Windows binaries before the Windows CI job is green and required
  (D1).
- Do not pay for an Apple Developer account for this (D6).
- Do not put a PAT in the workflow file — it goes in repository secrets, and
  only Phase 3 needs one.
- Do not publish to homebrew-core; use the tap (D6).
- Do not build a Docker image of the TUI (D9).
- Do not add `windows/arm64`, `linux/386`, `freebsd` or `riscv64` until someone
  asks for one by name.
- Do not remove `release.draft: true`. The human review of the changelog is the
  last check before a release is real.
- Do not let the install script `sudo` on the user's behalf (D4).
