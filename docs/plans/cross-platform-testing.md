# Plan: Test Cais on Windows and macOS for Free

> **Before you start.** Work on a feature branch of small commits, merged
> `--no-ff`; `go build ./... && go vet ./... && go test ./... && gofmt -l .`
> green at **every** commit, not just at the tip — `docs/ROADMAP.md`
> §Conventions is the full contract and `CONTRIBUTING.md` explains how a TUI
> gets tested. Behaviour that only shows on screen gets checked in the real app
> with VHS before it is committed. **Step 6 of the post-alpha order,** before the release work rather than after it.

The ask (from a discovery session): *"Is there a way I can test this tool on a
Windows and a Mac machine for free, using some kind of online testing
platform?"* The plan below is the researched answer, written down.

## Status of the ask — what "test" means for a TUI

The platforms people usually mean by "online testing" — BrowserStack,
LambdaTest, Sauce Labs — test **websites in browsers**. Cais is a
terminal app. Its tests don't need a browser, and its runtime needs a real
terminal. So the answer splits into three distinct kinds of "testing", and the
free option differs per kind:

1. **Automated: build + test suite on real Windows and macOS machines.** The
   free, correct answer is **CI runners**. This repo is on GitHub and public,
   so **GitHub Actions is free and unmetered on standard runners**
   (docs.github.com, retrieved 2026-07-31:
   <https://docs.github.com/en/billing/concepts/product-billing/github-actions>).
   Today CI runs only on Ubuntu; adding `windows-latest` and `macos-latest` to
   the existing matrix costs $0 and needs no new accounts.
2. **Visual: proof the TUI actually renders on each OS.** VHS (already used
   for the README demo) can record the app booting and taking screenshots,
   run inside CI on the same runners.
3. **Hands-on: a human driving the TUI interactively on those OSes.** No good
   free online option exists for macOS (no free cloud Mac VM — AWS EC2 Mac is
   ~$1/hr+, MacStadium is paid or OSS-application-gated). Windows has free
   cloud VMs (AWS Free Tier t3.micro, 750 hrs/month for 12 months; Azure free
   credit) but they need a credit card and are a poor substitute for CI
   coverage. **Recommendation: do 1 and 2 in CI; skip 3.** That answers the
   real question — "does it work on Windows and macOS" — without a credit card
   or a human at a keyboard.

## Current state (measured, 2026-07-31)

- `.github/workflows/ci.yml` runs one `check` job on `ubuntu-latest`:
  `go build ./...`, `go vet ./...`, a gofmt listing check, `go test -race ./...`.
- `.goreleaser.yaml` builds `linux` and `darwin` (amd64 + arm64), and carries
  this explicit decision:

  ```yaml
  # No Windows build: the app shells out to `docker compose` and hands the
  # terminal to $EDITOR, neither of which has been tried there.
  ```

  That comment is the whole reason this plan exists: the two Windows risks are
  **untried**, not **proven fine** — and "untried" can be fixed by CI.
- `CONTRIBUTING.md` (Releases section): "GoReleaser ... builds for linux and
  darwin (amd64 and arm64)". The docs mirror the current, Windows-less state.
- `demo/demo.tape` and `demo/screenshots.tape` exist — VHS is already part
  of the project's workflow. `demo.tape`'s header records that VHS "needs
  ttyd + ffmpeg too".

## Runner facts this plan relies on (verified 2026-07-31)

- **GitHub Actions, public repo: free, unmetered.** Private repos get 2,000
  included minutes/month, Windows billed ~2× and macOS ~10× Linux-equivalent
  (Windows 2-core $0.010/min, macOS $0.062/min vs Linux $0.006/min —
  <https://docs.github.com/billing/reference/actions-runner-pricing>).
- **`windows-latest` (→ Windows Server 2025 image) ships Docker 29.1.5 and
  Docker Compose 2.40.3** (runner-images `Windows2025-Readme.md`, retrieved
  2026-07-31). A `docker compose` smoke test is possible on Windows runners.
- **macOS runners do not ship Docker** — a dedicated marketplace action,
  [setup-docker-on-macos](https://github.com/marketplace/actions/setup-docker-on-macos)
  (Colima-based), exists precisely because it is not preinstalled. macOS
  testing therefore covers the app but not the docker calls, unless we install
  it (adds minutes; see Alternatives).
- **`macos-latest` is arm64** as of 2026 (macos-26 arm64 GA 2026-02-26;
  `macos-latest` migration began 2026-06-15 — GitHub changelog, retrieved
  2026-07-31). This means macOS CI covers the `darwin/arm64` release artifact
  on real Apple Silicon. Intel macOS (`darwin/amd64`) only has `macos-13`
  while it lasts — decide whether that matters (probably not for this app).

## Solution

Three changes, ordered so each de-risks the next. Steps 1 and 3 are the core;
step 2 is cheap proof.

### Step 1 — CI matrix: run the existing checks on Windows and macOS

Change the `check` job in `.github/workflows/ci.yml` to a matrix:

```yaml
jobs:
  check:
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-latest, windows-latest, macos-latest]
    runs-on: ${{ matrix.os }}
    # ... existing steps unchanged ...
```

Details that matter:

- **Keep the steps identical across platforms.** Same build, vet, gofmt, and
  `go test -race ./...`. The test suite (component tests, ansi.Strip render
  assertions, the e2e rig driving a real `tea.Program` against an in-memory
  buffer) needs no terminal and no Docker, so it runs anywhere Go runs.
- **The gofmt step needs `shell: bash` on Windows.** It uses
  `unformatted=$(gofmt -l .)` — POSIX syntax. Windows runners default to
  pwsh; GitHub's Windows images ship Git-Bash, so declaring `shell: bash` on
  that step (or the whole job) makes it portable. This is the one line the
  matrix breaks; the plan calls it out so the fix is in the first commit.
- **Windows and macOS jobs are required, not `continue-on-error`.** That is
  the point: a platform regression should block main, like any other.
- **Don't pin runner labels yet.** Ride `-latest`; pin only if an image
  migration causes pain (both `macos-latest` and `windows-latest` migrated
  mid-2026; the migration churn is settling).
- **No Docker in this step.** The suite mocks the docker calls. Docker
  smoke-testing is Step 2.

### Step 2 — prove it boots and renders on each OS

Two sub-parts, cheap first:

1. **Boot smoke test (in the matrix, no new tooling).** Build the binary, run
   it for ~3 seconds against a fixture compose file, and assert it exits
   cleanly when terminated. On the Linux/macOS runners, wrap it in `script`
   (or use `timeout`) to give it a PTY — a Bubble Tea program panics or exits
   oddly without one, and *that* is precisely the class of platform bug we are
   hunting. On Windows, `echo q | cais` or a short `timeout` run is enough.
   This is the minimal "does it launch" proof.
2. **VHS screenshots (optional, nicer).** A small tape in `demo/` (e.g.
   `boot-smoke.tape`) that runs the built binary against a fixture, presses a
   key, and takes a `Screenshot`. The repo already documents VHS as needing
   ttyd + ffmpeg (`demo/demo.tape` header) — on CI that means installing VHS
   plus ttyd/ffmpeg per runner (choco/brew). That is ~1–2 min per job of
   install cost. **Decision: start with the boot smoke test; add VHS
   screenshots only if the boot smoke is not enough evidence.** The e2e rig in
   `src/model/rig_test.go` already proves rendering at the model level; VHS
   adds the *pixels*, not the logic.
3. **Docker smoke test (Windows only, cheap):** `windows-latest` ships Docker
   + Compose (verified above). A single `docker compose config`/`up -d` call
   against a fixture service exercises the exact `docker compose` invocation
   the app makes, on the OS the old goreleaser comment was worried about.
   macOS: skip unless we install Docker there (see Alternatives).

### Step 3 — Windows release builds (only after 1 and 2 are green)

The goreleaser comment is a *risk register*, not a law: it exists because the
Windows paths were untried. Once CI runs the suite and a docker smoke test on
real Windows, the evidence exists to revisit it.

- Add `windows` to `goos` in `.goreleaser.yaml` (amd64 + arm64). `CGO_ENABLED=0`
  is already set, so the ubuntu runner in `release.yml` cross-compiles Windows
  binaries with **no change to release.yml**.
- **Windows archives should be zip, not tar.gz.** GoReleaser v2 does this via
  per-format override (`format_overrides: - goos: windows, formats: [zip]`);
  tar.gz is not natively openable on Windows without extra tools.
- Rewrite the `# No Windows build:` comment to record what replaced it: the
  two named risks, now tested — docker compose on Windows runners (Docker
  Desktop ships compose v2 on user machines), and `$EDITOR` (Windows default
  is notepad; the user's chosen editor is their own concern).
- Update `CONTRIBUTING.md` (Releases paragraph: "linux and darwin" →
  "linux, darwin and windows") and add a one-line Windows requirement note to
  README's Requirements section (Docker Desktop with the compose plugin —
  same requirement as macOS/Linux, just stated).
- **Blast radius of Step 3:** `.goreleaser.yaml`, `CONTRIBUTING.md`, `README.md`.
  `release.yml` untouched.

## Alternatives considered

1. **Do nothing.** Cost: platform regressions land untested; Windows users
   cannot install the app; the "No Windows build" comment stays an untested
   fear. The ask was explicitly to test on both OSes — do-nothing fails it.
2. **Online VM platforms (BrowserStack/LambdaTest/Sauce Labs).** Free tiers
   are time-boxed trials and they test browsers, not terminals. A TUI inside
   their browser VM is a worse simulation than a real hosted runner. Rejected.
3. **Free cloud VMs (AWS Free Tier Windows, Azure credit).** Real Windows
   desktop you can RDP into and drive the TUI by hand — the only hands-on
   interactive option, and it needs a credit card plus a 12-month timer. There
   is **no free cloud macOS equivalent at all**. Not CI; complementary at most.
   Rejected for this plan; the "hands-on" itch is satisfied by the boot smoke
   test + VHS.
4. **A second CI system (Codemagic's 500 free macOS M2 min/month; Azure
   Pipelines' 1,800 min/month).** Same capability as the Actions matrix, with
   a second CI to maintain. Pointless while the repo is public (Actions is
   unmetered there). Keep as the fallback *if the repo ever goes private* and
   macOS minutes matter.
5. **MacStadium / CircleCI OSS programs.** Real free macOS hardware, but
   application-gated for established open-source projects. Overkill here.
   Footnote only.

## Who decides / blockers

- **No external sign-off.** All three steps are repo-local, use no new
  accounts, no money, no secrets. The only external dependency (public GitHub
  repo) already holds.
- **Four decisions, all taken** — they were once listed as maintainer calls
  with a recommendation each; the recommendation is now the instruction:
  Windows and macOS jobs are **required**, not `continue-on-error`; the
  `-latest` runner labels are **ridden, not pinned**; **boot-smoke first**, VHS
  screenshots only if the smoke test proves insufficient; Windows archives are
  **zip**.

## Blast radius per step

| Step | Files touched | Other effects |
|---|---|---|
| 1 — CI matrix | `.github/workflows/ci.yml` | PR checks grow by 2 OS runs; gofmt step gains `shell: bash`; `CONTRIBUTING.md` "CI runs exactly these" line stays true (same checks, more platforms) |
| 2 — boot smoke | `ci.yml` (new steps), maybe `demo/boot-smoke.tape` | ~2–3 s per job; VHS path adds install cost per runner |
| 3 — Windows builds | `.goreleaser.yaml`, `CONTRIBUTING.md`, `README.md` | Next release ships Windows archives; draft-release review in the release workflow is unchanged |

## Risks / unknowns to confirm during implementation

- **`go test -race` on Windows** is supported for amd64, but the e2e rig is
  timing-based (per CONTRIBUTING.md); its `WaitFor` loops may be flaky on a
  slower Windows runner. First matrix run reveals it; fix by loosening waits,
  not by dropping `-race`.
- **gofmt on Windows**: pure-Go tool, fine; only the *shell* around it needed
  the `shell: bash` fix.
- **VHS on Windows**: the repo's own tape header claims ttyd + ffmpeg are
  needed; verify whether a screenshot-only tape avoids ffmpeg before spending
  the install time. Fallback is the boot smoke test.
- **Docker on user Windows machines**: Docker Desktop bundles compose v2, so
  the CLI path the app uses is the same one CI exercises on the runner.
- **Intel macOS**: `darwin/amd64` loses real coverage when `macos-13` retires.
  Acceptable for this app; note it, don't fix it.

## Acceptance criteria

1. A PR runs the full check suite on ubuntu, windows, and macos, all green,
   with Windows/macOS required (no `continue-on-error`).
2. The gofmt check passes on the Windows runner (`shell: bash` fix in place).
3. A boot smoke test launches the built binary on each OS with a PTY where
   needed and exits cleanly; a `docker compose` smoke call runs on Windows.
4. A `goreleaser` dry-run (or the next tagged release) produces
   windows/amd64 + windows/arm64 zip archives alongside the linux/darwin ones.
5. `CONTRIBUTING.md` and the goreleaser comment describe the actual state; no
   stale "linux and darwin" claims anywhere (sweep for `darwin` after the
   edit).
6. Total cost to the project: $0.

## Implementation order

1. Step 1: CI matrix (one commit; the highest value and the smallest diff).
2. Step 2: boot smoke in the matrix; VHS tape only if wanted.
3. Step 3: goreleaser windows + docs, **after** 1 and 2 are green on a PR.
4. Verify: push the matrix PR and watch all three OSes go green; tag a release
   and check the Windows artifacts in the draft.

## Do not

- Do not add a second CI system while the repo is public.
- Do not sign up for credit-card free tiers (AWS/Azure VMs) for CI — they are
  for hands-on testing only, and macOS has none anyway.
- Do not add BrowserStack/LambdaTest/Sauce Labs.
- Do not change `release.yml` — the ubuntu runner cross-compiles Windows fine.
- Do not pre-emptively special-case `$EDITOR` or docker invocations for
  Windows in code; CI exists to *discover* those problems. If the matrix
  surfaces a real Windows bug, that fix is a follow-up plan, not part of this
  one.
