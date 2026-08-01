# Plan: Say What Is Wrong With Docker, Precisely

> **Before you start.** Work on a feature branch of small commits, merged
> `--no-ff`; `go build ./... && go vet ./... && go test ./... && gofmt -l .`
> green at **every** commit, not just at the tip — `docs/ROADMAP.md`
> §Conventions is the full contract and `CONTRIBUTING.md` explains how a TUI
> gets tested. Behaviour that only shows on screen gets checked in the real app
> with VHS before it is committed.

Feature request (discovery session, nothing implemented): *"Detect if docker is
present on the machine. If not, think about the best way to deal with it.
Either instruct the user how to install it, or install it automatically if
possible."*

**Decided with the owner (2026-08-01): diagnose and instruct. The app never
changes the machine.** The reasoning is in D2, and it is worth keeping, because
"just run the install script" will be proposed again by someone.

## Status — what a broken docker looks like today

There is no detection at all. Every docker path in the app is
`exec.Command("docker", …)` followed by `CombinedOutput()`, and every failure
becomes the same shape of message:

```go
return "", fmt.Errorf("docker compose ps failed: %w: %s", err, string(output))
```

That reaches the user through the error banner or the error modal. So the five
distinct, differently-fixed problems below all arrive as one line of
`exec.ExitError` noise, and the app carries on offering `s`, `t`, `r`, `p` and
`x` — every one of which will fail the same way, one keypress at a time.

The specific first-run failure worth picturing: a stranger installs the binary,
runs `stitch` in their homelab directory, and their daemon is not running.
They see their compose file parsed and their groups listed — the app looks like
it works — and then `s` prints
`docker compose ps failed: exit status 1: failed to connect to the docker API`.
Nothing tells them that `sudo systemctl start docker` is the whole fix.

## Research — measured on 2026-08-01

Engine 29.6.0, Compose v5.1.4, Linux, socket `srw-rw---- root:docker`.

### The five states, and how to tell them apart

| # | State | How it is detected | What the user must do |
| --- | --- | --- | --- |
| 1 | `docker` not installed | `exec.LookPath("docker")` returns `exec.ErrNotFound` | install docker |
| 2 | Compose V2 plugin absent | `docker compose version --short` exits non-zero | install the compose plugin |
| 3 | Daemon not running | `docker version --format {{.Server.Version}}` exits non-zero, output does **not** mention permission | start the daemon |
| 4 | Socket permission denied | same probe, output mentions `permission denied` | add the user to the `docker` group |
| 5 | Fine | all three succeed | nothing |

A sixth is not a failure but is worth reporting inside states 3 and 4: the
**active context / `DOCKER_HOST`**. `docker context show` prints `default` on
this machine; a user who once ran `docker context use desktop-linux` and
uninstalled Docker Desktop gets state 3 forever with a perfectly healthy daemon
running underneath. Printing the endpoint being dialled is the difference
between a five-minute fix and an afternoon.

### Exact strings and exit codes

Captured, not invented — the parser fixtures lesson in `TODO.md` (the HEALTH
column shipped broken for months because its fixtures were written from the Go
struct instead of from docker's output) applies to error text too:

```
$ DOCKER_HOST=unix:///tmp/nope.sock docker version --format '{{.Server.Version}}'
failed to connect to the docker API at unix:///tmp/nope.sock; check if the path
is correct and if the daemon is running: dial unix /tmp/nope.sock: connect: no
such file or directory
                                                                    (exit 1)

$ docker frobnicate
docker: unknown command: docker frobnicate                          (exit 1)
```

Docker **28 and earlier** worded state 3 differently:
`Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?`
Both spellings are in the field right now, which is the argument for D1.

State 4's marker is the substring `permission denied` (the full message is
`permission denied while trying to connect to the Docker daemon socket at …`).
It could not be reproduced on this machine — the author is in the `docker`
group — so it is the one string in this plan taken from Docker's documentation
rather than from a terminal. Treat it as a heuristic (D1), not as a contract.

### Timings

| Probe | Cost |
| --- | --- |
| `exec.LookPath("docker")` | microseconds, no process |
| `docker compose version --short` | ~40 ms |
| `docker version --format '{{.Server.Version}}'` | **27 ms** (hits the daemon) |
| `docker info --format '{{.ServerVersion}}'` | 162 ms |

The whole preflight is under 100 ms, which is why D3 can afford to run it at
startup *and* on every error path.

### The distro question

`/etc/os-release` on the author's own machine:

```
ID=zorin
ID_LIKE="ubuntu debian"
```

This is the case that breaks the obvious implementation. A remediation table
keyed on `ID` alone has no entry for `zorin`, `pop`, `linuxmint`, `raspbian`,
`endeavouros` or half the desktop Linux world, and would fall through to the
generic message **on the machine this app is developed on**. `ID_LIKE` is a
space-separated list and must be walked after `ID` misses.

## Solution overview

Three pieces:

1. **`utils.DockerStatus`** — a probe (impure, three commands) split from a
   **classifier** (pure, fully unit-testable without docker installed).
2. **A diagnosis modal** that says which state was found, what it means, and
   the exact command that fixes it on *this* machine — copyable, never run.
3. **Wiring**: probe once at startup and open the modal if it fails; re-probe
   before reporting any docker error, so a raw `exec.ExitError` never reaches
   the user when a diagnosis is available.

No new keybinding (D5).

## Design decisions

### D1. Probe, don't parse — and confine the one heuristic

Classification comes from *which probe failed*, not from what it printed. Only
the split between states 3 and 4 needs the output at all, and that split is a
single `strings.Contains(out, "permission denied")` with a documented fallback:
anything unrecognised is state 3, whose remediation text ends with "if the
daemon is running, check the socket's permissions" so the fallback still points
at state 4.

This is what keeps the feature from rotting. Docker rewords its errors between
majors (measured above: 28 → 29 changed state 3's entire sentence), and a
classifier built on `strings.Contains` of the whole message would degrade
silently into "unknown error" the next time. Exit codes and probe ordering do
not reword.

### D2. The app never installs, starts, or configures anything

The ask offered automatic installation. **No** — and the reasons deserve to be
written down once, here, rather than re-argued:

1. **It needs root.** A TUI holding the alternate screen cannot prompt for a
   sudo password without suspending itself, and a program that asks for your
   root password to fix a problem it just diagnosed is indistinguishable in
   shape from the thing you should never type your root password into.
2. **There is no single correct install.** apt with Docker's repo, dnf, pacman,
   Docker Desktop, colima, OrbStack, rootless, Snap (which breaks compose file
   paths outside `$HOME`), and the distro's own `docker.io` package (which on
   Debian and Ubuntu ships *without* the compose plugin — state 2, the second
   most common failure in this list, is caused by exactly that package). The
   app cannot pick for the user, and picking wrong leaves two docker
   installations on the machine.
3. **The one-liner is not recommended by its own author.** `get.docker.com` is
   documented by Docker as unsuitable for production; it adds apt sources and
   GPG keys and pins a channel. Running it from inside a compose viewer is a
   larger change to the machine than everything else this app does put
   together.
4. **macOS cannot be scripted honestly.** It means a GUI installer with
   licensing terms, or a third-party runtime the user has opinions about.
5. **It contradicts what the app is.** The README's pitch is "nothing extra to
   host, nothing listening on a port, and no state of its own". A tool that
   installs system packages is a different, scarier tool, and the audience most
   likely to try it — someone SSH'd into a server they care about — is the
   audience least likely to forgive it.

The same reasoning rules out the softer version (offer to run `sudo systemctl
start docker` behind a confirm). It was considered and declined with the owner
on 2026-08-01: the diagnosis plus an exact command is 95% of the value at 0% of
the blast radius, and the user retains the thing they should retain, which is
the decision to change their machine.

**What the app does instead:** prints the command it would have run, selectable
by the terminal and copyable, and gets out of the way.

### D3. Non-blocking. The app is still useful without docker

Do not gate startup. Without a daemon the app can still resolve, parse,
display, highlight, browse, group-tag and edit the compose file — which is more
than half of what it is for, and is exactly what someone writing a stack on a
laptop before deploying it wants.

So:

- Probe in `Init`, as a `tea.Cmd` alongside the existing three. It is 100 ms,
  but it is still a subprocess and belongs off the update loop.
- If the result is state 5, nothing happens and nothing is shown. **A healthy
  machine must see no new UI whatsoever.**
- Otherwise open the diagnosis modal, subject to the same guard the bootstrap
  modal learned the hard way (`TODO.md`, the flaky-bootstrap entry): **only
  when no other modal owns the screen.** A background result has no business
  closing a modal the user is working in.

### D4. Every docker error re-probes before it is reported

The startup probe answers "was docker healthy when we started". Daemons stop,
sockets get replaced by an upgrade, laptops sleep. So `reportForegroundError`
(and the banner path beside it) gains a check: when the error came from a
docker call, re-run the preflight; if it now reports a state other than 5,
show the diagnosis modal instead of the raw error.

This is what makes the feature worth its size. Without it, the diagnosis is a
startup nicety; with it, it is the app's permanent answer to "docker broke",
and the raw `exec.ExitError` string stops being user-facing for the four
diagnosable states.

Cost control: only on the error path, and only for docker errors — never on the
5-second poll's success path. A poll failing every 5 seconds re-probes every 5
seconds, which is 100 ms of subprocess per poll; acceptable, and it means a
daemon coming back is noticed by the banner clearing itself.

### D5. No new keybinding

The modal closes on `esc` / `q` (the overlay contract every modal in the app
already answers). It is reopened by *doing something that needs docker* — which
is the moment the user wants it, and is D4.

A dedicated "show docker status" key was considered and dropped: `keys.Global`
has eight entries and the footer is width-constrained. One verb, one binding —
and this verb's trigger is "an action failed", not "a key was pressed".

### D6. Compose V1 gets a sentence of its own

If state 2 is detected and `exec.LookPath("docker-compose")` **succeeds**, the
message says so explicitly:

> `docker-compose` (V1) is installed, but Stack Stitcher needs the V2 plugin
> (`docker compose`, no hyphen). V1 reached end of life in July 2023 and does
> not support `--format json`.

That is a real and frequent confusion — the two spellings look like the same
tool — and the app's entire container-status path depends on `--format json`,
which V1 never had. Diagnosing it costs one `LookPath`.

### D7. Remediation text lives in one table, keyed by state × platform

```go
// Remedy is what the user should do about a DockerStatus on this machine:
// one sentence of what is wrong, and the exact command that fixes it.
type Remedy struct {
    Summary string   // "The Docker daemon is not running."
    Steps   []string // shell lines, printed verbatim, never executed
    Note    string   // optional: the Snap/context/V1 caveats
    DocsURL string
}

func RemedyFor(status DockerStatus, host HostInfo) Remedy
```

`HostInfo` is `{GOOS string; DistroID string; DistroLike []string}`, built by a
tiny `utils.DetectHost()` that reads `runtime.GOOS` and, on Linux, parses
`/etc/os-release`. **Match `ID` first, then each entry of `ID_LIKE` in order,
then fall back to the generic entry** — see the Zorin case in the research
section; the author's own machine is the regression test for this.

Platform families to cover, and nothing more:

| Family | Matches | State 1 remedy |
| --- | --- | --- |
| debian | `debian`, `ubuntu` (via `ID_LIKE` this reaches mint, pop, zorin, raspbian, …) | the four `apt-get` lines from Docker's official repo instructions, or a link |
| fedora | `fedora`, `rhel`, `centos` | `sudo dnf install docker-ce docker-ce-cli containerd.io docker-compose-plugin` |
| arch | `arch` | `sudo pacman -S docker docker-compose` |
| suse | `opensuse`, `suse` | `sudo zypper install docker docker-compose` |
| darwin | `runtime.GOOS == "darwin"` | Docker Desktop, or `brew install colima docker docker-compose` + `colima start` |
| generic | anything else | link to `https://docs.docker.com/engine/install/` |

**Do not embed the full multi-line apt repo dance.** It is six commands, it
changes, and getting it wrong is worse than not printing it. Print the one line
that installs the plugin when the engine is already there (state 2:
`sudo apt-get install docker-compose-plugin`) and link the official page for a
full install. The states the app can fix in one line — 2, 3, 4 — are the states
worth printing commands for, and they are also the three most common.

State 3 and 4 remedies are the same everywhere on Linux, so they do not need
the distro table at all:

- **3:** `sudo systemctl start docker` (and `sudo systemctl enable --now docker`
  to survive a reboot); on macOS, "start Docker Desktop" / `colima start`.
- **4:** `sudo usermod -aG docker $USER`, followed by the part everyone
  forgets — **the group is picked up at next login**; `newgrp docker` in the
  current shell as the immediate workaround. Add one sentence noting that group
  membership is root-equivalent, and that rootless mode is the alternative,
  with the link. Say it once, do not lecture.

## Detailed changes

### 1. `src/utils/DockerPreflight.go` (new)

```go
type DockerState int

const (
    DockerOK DockerState = iota
    DockerNotInstalled
    DockerComposeMissing
    DockerDaemonUnreachable
    DockerPermissionDenied
)

// DockerStatus is the whole answer: the state, plus the facts worth
// reporting alongside it.
type DockerStatus struct {
    State          DockerState
    EngineVersion  string // "29.6.0", when known
    ComposeVersion string // "5.1.4",  when known
    Endpoint       string // DOCKER_HOST or the active context's endpoint
    ComposeV1Found bool   // docker-compose (hyphen) is on PATH
    Raw            string // the failing probe's output, for the modal's detail line
}

// probes is the seam that makes this testable. The real one shells out; the
// tests supply a struct literal.
type probes struct {
    lookPath       func(string) (string, error)
    composeVersion func() (string, error)
    engineVersion  func() (string, error)
    endpoint       func() string
}

// classify is pure: given probe results, which state is this?
func classify(p probes) DockerStatus

// DockerPreflight runs the real probes and classifies them.
func DockerPreflight() DockerStatus
```

The split is the point. `classify` has no `exec` in it, so the whole decision
table is unit-tested on a machine where docker is missing, present, broken or
irrelevant — including the states that cannot be reproduced locally (state 4).

### 2. `src/utils/HostInfo.go` (new)

`DetectHost()` and `RemedyFor(status, host)`. `/etc/os-release` parsing is
twenty lines: split on `=`, strip quotes, keep `ID` and `ID_LIKE`. A missing or
unreadable file is not an error — it yields the generic family.

### 3. `src/cmds/CheckDocker.go` (new)

```go
type DockerStatusMsg struct{ Status utils.DockerStatus }

func CheckDocker() tea.Cmd // runs the preflight off the update loop
```

### 4. `src/components/dockerstatusmodal/` (new package)

`Model.go` and `View.go` — no `Update.go` needed at first; it answers two keys.
Follow `aboutmodal` exactly: a read-only overlay, `esc`/`q` close it via
`cmds.CloseModal(nil)`. It renders, from top:

- a title pill in `StatusError` (or `StatusStarting` for state 4, which is a
  configuration problem rather than an outage),
- the summary sentence,
- the endpoint being dialled, dimmed, when the state is 3 or 4,
- the command block — `Steps`, one per line, in the theme's code styling,
- the note and the docs URL, dimmed.

Colors come from `appstyles.Active`. No raw hex, and no `var` built at package
init (see `docs/DESIGN.md` §*Color lives on a Theme*).

### 5. `src/model/Init.go`

Add `cmds.CheckDocker()` to the batch.

### 6. `src/model/Update.go`

- Handle `cmds.DockerStatusMsg`: store it on `AppModel`; if the state is not
  `DockerOK` **and** `activeModal == nil`, open the diagnosis modal.
- In `reportForegroundError` and the banner path: when the error text is from a
  docker call, dispatch `cmds.CheckDocker()` and let the handler above decide.
  Keep the existing asymmetry between the modal and banner paths intact — the
  comment in `TODO.md` explaining `lastErrorFromPoll` is load-bearing and this
  change must not disturb it.

### 7. `README.md`

The **Requirements** line currently says "Docker with the Compose plugin on
your `PATH`". Add one sentence: if something is missing or the daemon is not
running, the app says which of the five it is and what to type. That is a
selling point for the audience that has been bitten by `docker.io` shipping
without the plugin.

### 8. `docs/DESIGN.md`

A short subsection under §5 — *Docker's absence is a diagnosis, not an error* —
carrying D1 (probe, don't parse), D2 (never change the machine, with the five
reasons compressed to two sentences), and D4 (errors re-probe).

## Tests

### `src/utils/DockerPreflight_test.go`

Table-driven over `classify`, with injected probes. One case per state, plus:

| case | probes | want |
| --- | --- | --- |
| healthy | all succeed | `DockerOK`, versions populated |
| no binary | `lookPath` → `exec.ErrNotFound` | `DockerNotInstalled` |
| plugin missing | binary ok, `composeVersion` → exit 1 | `DockerComposeMissing` |
| plugin missing, V1 present | as above + `docker-compose` on PATH | `DockerComposeMissing`, `ComposeV1Found: true` |
| daemon down | engine probe → "failed to connect to the docker API…" | `DockerDaemonUnreachable` |
| daemon down, old wording | engine probe → "Cannot connect to the Docker daemon…" | `DockerDaemonUnreachable` |
| permission denied | engine probe → "permission denied while trying to connect…" | `DockerPermissionDenied` |
| unrecognised failure | engine probe → "some future wording" | `DockerDaemonUnreachable` (the documented fallback) |

The two daemon-down wordings are the test that pins D1: both must classify the
same, and neither may be matched by a substring of the other.

### `src/utils/HostInfo_test.go`

`parseOSRelease` against captured file contents:

- Zorin (`ID=zorin`, `ID_LIKE="ubuntu debian"`) → debian family. **This is the
  regression test for the whole ID_LIKE decision; name it so.**
- Ubuntu (`ID=ubuntu`, `ID_LIKE=debian`) → debian family.
- Fedora (`ID=fedora`, no `ID_LIKE`) → fedora family.
- Arch (`ID=arch`) → arch family.
- Empty / missing file → generic family.
- `ID_LIKE="rhel centos fedora"` → fedora family (first match in the list wins).

And `RemedyFor`: every (state, family) pair returns a non-empty `Summary`, and
no `Steps` entry contains a `|` pipe into a shell or the string `get.docker.com`
— a test that pins D2 against a future "helpful" edit.

### `src/model/`

One rig test (`src/model/dockerstatus_test.go`, mirroring `about_test.go`):
feeding `DockerStatusMsg{State: DockerDaemonUnreachable}` opens the modal;
feeding `DockerOK` opens nothing; feeding a failure while another modal is open
opens nothing and leaves the existing modal in place.

That last case is the bootstrap-flakiness lesson, pinned.

## Edge cases and unknowns

- **Docker in a context that is not `default`.** `docker context show` costs a
  process; `DOCKER_HOST` is free. Read the env var first, fall back to the
  context command only when the env var is empty *and* the state is 3 or 4 (so
  a healthy machine never pays for it).
- **Rootless docker** puts the socket at `$XDG_RUNTIME_DIR/docker.sock` and
  needs no group membership. It classifies as state 5 when working; when not,
  it looks like state 3. The remedy note for state 3 mentions rootless in one
  clause. Do not try to detect it.
- **Snap-installed docker** confines the daemon so compose files outside
  `$HOME` cannot be read. It classifies healthy and then fails at `up` with a
  permission error on a path. Out of scope; a sentence in the state-3 note is
  the most that is honest.
- **Windows.** `runtime.GOOS == "windows"` is not in D7's table because the app
  has no Windows build yet (`docs/plans/cross-platform-testing.md` is roadmap
  step 6). The generic family covers it; when Windows CI goes green, add the
  Docker Desktop entry there rather than guessing now.
- **A daemon that is up but wedged** (`docker version` returns, `docker compose
  ps` hangs) classifies as state 5 and the user gets the raw error. Correct:
  the app should not invent a diagnosis it cannot support.
- **`docker` present but not executable** (permissions on the binary) returns
  `exec.ErrPermission` from `LookPath`, not `ErrNotFound`. Classify it as state
  1 with the note "found at `%s` but not executable" rather than inventing a
  sixth state.

## Effort / gain

**One to one and a half days.** Two pure files with their tests (half a day),
one modal following an existing pattern (two hours), the wiring (two hours),
docs (an hour).

The gain is concentrated in the first run by a stranger, which is exactly the
moment `docs/plans/launch-and-outreach.md` is about. It also removes a whole
category of issue reports — "it doesn't work, here's a screenshot of `exit
status 1`" — that would otherwise arrive on launch day with no information in
them.

## Blast radius

- Startup gains one ~100 ms subprocess, off the update loop, whose result is
  discarded entirely when the machine is healthy.
- One new modal, opened only on failure, subject to the existing modal guard.
- `reportForegroundError` gains a branch. Touching it is the risk in this plan:
  it is the function `TODO.md` documents as having a load-bearing asymmetry
  between its modal and banner paths. Read that entry before editing, and do
  not change which path sets `lastErrorFromPoll`.
- No writes, no config, no new dependency, no new key.

## Do not

- **Do not install, start, enable, or `usermod` anything.** Not behind a
  confirm, not behind a flag, not "just for Linux". D2 is the decision; if it
  is ever revisited, it is revisited in `docs/DESIGN.md` with the owner, not in
  a pull request.
- **Do not pipe anything to a shell in the printed steps**, and do not print
  `curl … | sh`. The test in `HostInfo_test.go` enforces this.
- **Do not block startup or refuse to open the compose file.** Half the app
  works without a daemon and that half is the half that edits your file.
- **Do not classify on the full error message.** Exit codes and probe order,
  plus the one documented `permission denied` heuristic.
- **Do not add a keybinding** for the modal, and do not put a permanent "docker
  is down" element in the footer — the footer is already width-constrained
  enough to have needed its own shedding mechanism.
- **Do not re-probe on the poll's success path**, only on failures.
- **Do not invent error strings for the tests.** Every fixture in this plan was
  captured from a real docker; state 4's comes from Docker's documentation and
  is marked as such. If a state cannot be reproduced, say so in the test's
  comment rather than writing a plausible-looking string and asserting on it.

## Acceptance criteria

1. On a healthy machine, the app looks and behaves exactly as it does today —
   no modal, no banner, no extra line anywhere.
2. With the daemon stopped, the app still opens, parses and displays the
   compose file, and shows a modal reading "The Docker daemon is not running"
   with `sudo systemctl start docker`.
3. With `DOCKER_HOST` pointed at a socket that does not exist, the modal names
   that endpoint.
4. With `docker` renamed off `PATH`, the modal says docker is not installed and
   prints the remedy for the detected distro family — on the author's Zorin
   machine, the Debian family's, not the generic one.
5. Stopping the daemon while the app is running turns the next `s` into the
   diagnosis modal, not `exit status 1`.
6. `classify` has no `os/exec` import, and its tests pass with docker
   uninstalled.
7. Every existing test passes at every commit.
