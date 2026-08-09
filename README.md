# Stack Stitcher

**Run your self-hosted stack from the terminal, without leaving your compose file behind.**

[![CI](https://github.com/filipemolina/stack-stitcher/actions/workflows/ci.yml/badge.svg)](https://github.com/filipemolina/stack-stitcher/actions/workflows/ci.yml)
![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)
![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)
![Status](https://img.shields.io/badge/status-work%20in%20progress-orange)

<img src="./assets/logos/stack-stitcher-readme-banner.svg" alt="Stack Stitcher readme banner" width="100%" />

## Demo

![Stack Stitcher: starting a group, tailing its logs, editing a service](./demo/demo.gif)

Stack Stitcher is a keyboard-driven terminal UI for a self-hosted stack running on
Docker Compose. It reads your `compose.yml`, groups services the way you think about
them, and gives you one key each for start, stop, restart, pull, logs and edit.
Nothing extra to host, nothing listening on a port, and no state of its own.

Your compose file stays the source of truth. Every change Stack Stitcher makes is
written back into that file, comments and key order intact, so `docker compose`
on the command line and Stack Stitcher never disagree about what your stack is.

## What it does

**Groups, not a flat list of containers.** A group is a Compose `profiles:`
tag. Start all services in a group with one key, tail all their logs with another,
and see at a glance which are running, which are healthy, and on what ports.

![The Groups page: a group selected, its member services and their state](./demo/screenshot-groups.png)

**Everything you need to check, on one screen.** Ports, restart policy,
networks, volumes, `depends_on`, healthcheck, image reference and resource
limits, beside live memory, CPU, network and disk I/O for the running container.

![The Services page: one service's configuration and live runtime stats](./demo/screenshot-service.png)

**A working link to the thing you just started.** The config details view shows
a real hyperlink built from the port the service actually publishes and the
address your terminal is sitting behind. Ctrl-click it, or copy it with `y`.
The app never opens a browser itself.

**A healthcheck in one keypress.** `h` on a selected service opens a picker of
templates: Postgres, MariaDB, Redis, nginx, and others, each using a probe tool
that ships in the real image, plus a generic HTTP fallback. Enter writes a
validated `healthcheck:` block straight into the compose file.

**New services without leaving the terminal.** `n` on the Services page
searches Docker Hub live as you type. Official images are marked and star
counts shown. It confirms a name and image before writing a minimal
fragment straight into the compose file and opening the inline editor on it.

**Edit the compose file in place, as YAML.** `e` opens the service's own
fragment in an inline editor: real YAML, not a form, so every Compose field is
reachable. It validates as you type, auto-indents on Enter, indents with
`tab`/`shift+tab`, and refuses to write a fragment that would not parse as
Compose.

![The inline YAML editor open on a service, with live validation](./demo/screenshot-editor.png)

**Logs without leaving.** `l` streams `docker compose logs -f` for a service or
a whole group in an overlay, with follow mode and scrollback.

![Streaming logs for a service](./demo/screenshot-logs.png)

<details>
<summary>More features</summary>

**The compose file itself,** syntax-highlighted and scrollable. `E` opens it in
your `$EDITOR`; `b` browses the other compose files in the same directory and
switches which one the app is driving.

![The Files page: the loaded compose file with syntax highlighting](./demo/screenshot-files.png)

**Fourteen themes,** previewed live as you move the cursor: a light and dark
pair of built-in themes plus Catppuccin Mocha, Gruvbox Dark, Tokyo Night,
Nord, Dracula, Solarized Dark, One Dark, Everforest Dark, Rosé Pine and
Kanagawa Wave. `Enter` persists your choice.

![The theme picker, previewing a theme live over the Files page](./demo/screenshot-themes.png)

</details>

Also: create, rename, and delete groups; change which services belong to them;
confirm-guarded removes; a status re-poll every five seconds so panels reflect
changes made outside the app; and a `?` overlay listing every key, with the ones
that do nothing on the current screen dimmed.

## Install

```bash
go install github.com/filipemolina/stack-stitcher@latest
```

Or build from source:

```bash
git clone https://github.com/filipemolina/stack-stitcher.git
cd stack-stitcher
make build     # installs to $(go env GOPATH)/bin, usually ~/go/bin
```

There are no downloadable binaries yet. The release pipeline is built (a `v*`
tag builds Linux and macOS, amd64 and arm64) but nothing is tagged, so `go
install` or a clone is the way in for now.

**Requirements:** Docker with the Compose plugin on your `PATH`, a terminal, and
Go 1.26+ to build. If something is missing or the daemon is not running, the
app says which of the five it is and gives you the exact command to fix it.

## Use

```bash
stitch                                  # the compose file in this directory
stitch --dir ~/homelab/media            # resolve one in that directory
stitch --file ~/homelab/compose.prod.yml  # open exactly this file
```

With no flags it auto-detects `compose.yaml`, `compose.yml`,
`docker-compose.yaml`, `docker-compose.yml`, the same order Docker uses. The
file that wins is named in the footer, and it is passed as `--file` to every
`docker compose` call, so the commands always act on the file the panels
describe. In a directory with no compose file at all, the app offers to write
one.

### Keys

`?` lists every key in context. The ones worth knowing:

| Key | Action |
| --- | --- |
| `1` `2` `3` | Groups / Services / Files (`[` and `]` step through them) |
| `↑` `↓` `k` `j` | Move the cursor; the details panel follows it |
| `Tab` | Move focus between the list and the details panel |
| `s` `t` `r` `p` `x` | Start · Stop · Restart · Pull · Remove (`x` confirms first) |
| `l` | Stream logs for the service, or for every service in the group |
| `y` | Copy a service's URL (when it publishes one) |
| `h` | Add a healthcheck from the template picker |
| `e` | Edit: a service's YAML inline, or a group's membership |
| `E` | Open the whole compose file in `$EDITOR` |
| `n` `R` `d` | New (a group on the Groups page, a service on the Services page: search Docker Hub live, then confirm) · Rename group · Delete group |
| `/` | Filter the list by name |
| `T` `?` `a` `q` | Themes · Help · About · Quit |

Start/Stop/Restart/Pull/Remove run `docker compose` underneath, scoped to every
service in the group on the Groups page, to one service on the Services page.

## Status

Early, and honest about it. Everything shown above works today; what follows is
what does not, in the order it is being closed. The sequence and the reasoning
live in [docs/ROADMAP.md](docs/ROADMAP.md):

- **No `.env` surface.** Values are interpolated correctly; the file that holds
  them is not visible or editable in the app, and secrets are not masked.
- **Blank lines between services are not preserved** across a write. Comments,
  quoting and key order are. This is accepted rather than fixed: a blank line
  inside a block scalar (`command: |`) is part of the string, and silently
  rewriting your data is a worse failure than losing your spacing.
- **A narrow terminal shows less, not worse.** Under roughly 135 columns the
  keybinding bar starts dropping hints, and a narrow details panel drops
  columns from the member table, widest and least important first, and never
  `q quit`, `? help` or the service's own name.

Issues and ideas are welcome, and at this stage they still change the
direction.

## Built with

Go, [Bubble Tea](https://github.com/charmbracelet/bubbletea) /
[Lip Gloss](https://github.com/charmbracelet/lipgloss) for the UI, and
[compose-go](https://github.com/compose-spec/compose-go), the same parser
Docker itself uses, for the file. Docker actions shell out to the
`docker compose` CLI rather than binding the SDK, so what the app does is what
you would have typed.

## License

[MIT](LICENSE). © 2026 Filipe Molina.
