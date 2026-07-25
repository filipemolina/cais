# Stack Stitcher

> A fast, keyboard-driven terminal UI for managing your self-hosted Docker Compose services.

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)
![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)
![Status](https://img.shields.io/badge/status-work%20in%20progress-orange)

Stack Stitcher reads a Docker **Compose** file and turns it into an interactive TUI, so you can browse and operate the services in your homelab or self-hosted stack without memorizing `docker compose` commands. It parses your `compose.yml` with the same specification library Docker itself uses, and renders everything through [Charm](https://charm.sh)'s Bubble Tea and Lip Gloss.

> **New here?** Read [docs/DESIGN.md](docs/DESIGN.md) for the guiding principles
> (groups-first navigation, group vs profile terminology, the data model).
> It will save you from re-litigating decisions that are already made.

## Project status

Stack Stitcher is under **active development**. Compose parsing, navigation, starting/stopping services (individually or as a whole group), creating/deleting groups, streaming live logs, and bootstrapping a new compose file from inside the TUI all work. Editing existing services works by handing the YAML to your `$EDITOR` — `e` for one service, `E` for the whole file — rather than by filling in a form, so every compose field is reachable and your comments, quoting and key order are kept. (Blank lines between services are not: the YAML library preserves comments but not blank lines, so any write closes the spacing up.) Editing inline in the panel is still on the roadmap. See [TODO.md](TODO.md) for the current worklist and completed recent work. Feedback, issues, and ideas are genuinely welcome and help shape where it goes next.

![Stack Stitcher demo](./demo/demo.gif)

## Features

- **Reads standard Compose files.** Uses the official [`compose-go`](https://github.com/compose-spec/compose-go) parser, so it understands the same `compose.yml` your Docker setup already relies on — no custom config format to learn.
- **Keyboard-first TUI.** Built on [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles), and [Lip Gloss](https://github.com/charmbracelet/lipgloss) for a responsive, styled terminal experience.
- **Start/stop a whole group together.** Compose "profiles" group related services (e.g. everything a self-hosted app needs); Stack Stitcher lets you Start/Stop/Restart/Pull/Remove all of them in one keypress instead of remembering which services belong together.
- **Start/stop a single service.** The same five actions are available for one service at a time from the Dashboard view.
- **Stream live logs.** Press `l` on a focused service or group panel to open a full-screen overlay that tails `docker compose logs -f` in real time, with follow-mode and scrollback.
- **Automatically refreshed status.** Container state is rechecked every five seconds while a compose project is loaded and no modal is open, so status panels reflect changes made outside Stack Stitcher.
- **Full-height, context-aware layout.** The app fills the terminal with a pinned header (wordmark + tabs) and footer (keybinding bar); the body region stretches to use every available row. Tabs show user-facing labels such as **Groups** for Home and **Files** for Compose Files, while the underlying page IDs stay the same.

## Requirements

- **Go 1.26+** — to build from source.
- **Docker** with the Compose plugin available on your `PATH`.
- A Compose file describing your services: `compose.yaml`, `compose.yml`, `docker-compose.yaml`, or `docker-compose.yml`.

## Installation

Clone the repository and build the binary:

```bash
git clone https://github.com/filipemolina/stack-stitcher.git
cd stack-stitcher
make build
```

`make build` runs `go install .`, which installs the binary to
`$(go env GOPATH)/bin` (usually `~/go/bin`). Ensure that directory is on your
`PATH`; no `sudo` or manual move is needed.

```bash
command -v stack-stitcher
```

To run it during development without building:

```bash
make dev   # equivalent to: go run main.go
```

## Usage

Run Stack Stitcher from a directory that contains your Compose file:

```bash
stack-stitcher
```

It auto-detects the compose file in the current directory, checking in order: `compose.yaml`, `compose.yml`, `docker-compose.yaml`, `docker-compose.yml`. There's no flag to point at a file elsewhere yet — `cd` into the project directory first.

### Key bindings

Pages are switched with `Alt` plus the underlined letter in each nav tab — the
nav itself never takes keyboard focus, so `Tab` is free to move between the two
body panels. `Alt` rather than `Ctrl` because terminals intercept `Ctrl+S` as
flow control and `Ctrl+D` as end-of-input.

| Key | Action | Where |
| --- | --- | --- |
| `Alt`+`G` `D` `F` `S` | Jump to Groups / Dashboard / Files / Settings | Everywhere |
| `Tab` / `Shift+Tab` | Move focus between the two body panels | Everywhere |
| `Space` | Select the highlighted group or service | Groups/Services list focused |
| `s` | Start | A group or service panel focused |
| `t` | Stop | A group or service panel focused |
| `r` | Restart | A group or service panel focused |
| `p` | Pull | A group or service panel focused |
| `x` | Remove (asks for confirmation) | A group or service panel focused |
| `n` | Create a new group | Groups panel focused |
| `d` | Delete the highlighted group | Groups panel focused |
| `e` | Edit the service's YAML in `$EDITOR` | Service details panel focused |
| `E` | Edit the whole compose file in `$EDITOR` | Service details panel focused |
| `l` | View live logs (streaming overlay) | A group or service panel focused |
| `f` | Toggle follow (auto-scroll) | Logs overlay open |
| `↑`/`↓` `PgUp`/`PgDn` | Scroll logs | Logs overlay open |
| `Esc` | Close the logs overlay | Logs overlay open |
| `q` / `Ctrl+C` | Quit | Everywhere |

Start/Stop/Restart/Pull/Remove run `docker compose` under the hood — scoped to the selected group (every service tagged with it) on the Home page, or to just the selected service on the Dashboard page.

### UI overview

Stack Stitcher fills the terminal. The top bar shows the `▌ Stack Stitcher` wordmark and page tabs; the body stretches to use all remaining rows, and the keybinding bar at the bottom is context-aware (action hints are hidden until a group or service is selected).

On **Home** the body is a two-pane layout: the Groups list on the left and the Group Details panel on the right. The Group Details panel no longer renders the large ASCII logo. Instead it shows:

- **No groups yet:** a *Getting started* card explaining that groups are Compose profiles and how to create one.
- **Groups exist, none selected:** a *Select a group* card prompting the user to pick from the list.
- **Group selected:** a header card with the group name, a status pill (ALL RUNNING / MIXED / STOPPED), and a running/stopped/services summary, followed by a member-services table (status dot, NAME, IMAGE, STATE, HEALTH, UPTIME, PORTS). When nothing is running a "Press `s` to start." footnote appears, and Start/Stop/Restart/Pull/Remove action buttons are pinned at the bottom.

The ASCII logo asset is still kept in `src/constants/Branding.go` for a future About modal.

## Tech stack

- **Language:** Go
- **TUI:** Bubble Tea, Bubbles, Lip Gloss (Charm)
- **Compose parsing:** `compose-spec/compose-go`
- **Docker actions:** shells out to the `docker compose` CLI (no Docker SDK dependency)

## Project layout

```
.
├── main.go            # Entry point — starts the Bubble Tea program
├── src/
│   ├── model/         # Top-level Bubble Tea model (AppModel, Update, View, Init)
│   ├── components/    # Nested Bubble Tea models — one per panel (lists, details, buttons)
│   ├── cmds/          # Message types + the tea.Cmds that produce them
│   ├── apptypes/      # Shared data types (list items, docker container, pages)
│   ├── utils/         # Non-Bubble Tea logic (compose file loading, docker exec, parsing)
│   ├── appstyles/     # Lip Gloss colors/styles
│   └── constants/     # Layout widths, branding, focusable component list
├── demo/              # VHS script + recorded demo gif
├── docs/              # design guidance and completed historical plans
├── TODO.md            # current worklist and completed recent work
├── Makefile           # dev / build targets
├── go.mod
└── go.sum
```

## Development

```bash
make dev           # run locally
make build         # install to $(go env GOPATH)/bin

go build ./...     # compile every package
go vet ./...       # static checks
go test ./...      # test suite
```

Contributions, issues, and feature ideas are welcome.

## License

Released under the [MIT License](LICENSE). © 2026 Filipe Molina.
