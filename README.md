# Stack Stitcher

> A fast, keyboard-driven terminal UI for managing your self-hosted Docker Compose services.

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)
![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)
![Status](https://img.shields.io/badge/status-work%20in%20progress-orange)

Stack Stitcher reads a Docker **Compose** file and turns it into an interactive TUI, so you can browse and operate the services in your homelab or self-hosted stack without memorizing `docker compose` commands. It parses your `compose.yml` with the same specification library Docker itself uses, and renders everything through [Charm](https://charm.sh)'s Bubble Tea and Lip Gloss.

## Project status

Stack Stitcher is under **active development**. The foundations are in place — Compose parsing, the service view, and navigation — while several actions and edge cases are still being built out and refined. It's ready to explore and experiment with, but not yet feature-complete, so expect some rough edges and occasional breaking changes as it works toward a stable release. Feedback, issues, and ideas are genuinely welcome and help shape where it goes next.

<!-- TODO: drop a screenshot or a short VHS/asciinema GIF here — a TUI sells itself visually. -->

## Features

- **Reads standard Compose files.** Uses the official [`compose-go`](https://github.com/compose-spec/compose-go) parser, so it understands the same `compose.yml` your Docker setup already relies on — no custom config format to learn.
- **Keyboard-first TUI.** Built on [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles), and [Lip Gloss](https://github.com/charmbracelet/lipgloss) for a responsive, styled terminal experience.
- **Fuzzy search.** Jump to any service in a large stack by typing part of its name.
- **Clipboard support.** Copy service details straight to your clipboard.
<!-- TODO: List the exact actions Stack Stitcher performs on a service (e.g. up / down / restart / view logs / status) and adjust the two lines above to match what's actually implemented. -->

## Requirements

- **Go 1.26+** — to build from source.
- **Docker** with the Compose plugin available on your `PATH`.
- A **`compose.yml`** (or `docker-compose.yml`) describing your services.

## Installation

Clone the repository and build the binary:

```bash
git clone https://github.com/filipemolina/stack-stitcher.git
cd stack-stitcher
make build
```

This produces the binary at `dist/stack-stitcher`. Move it somewhere on your `PATH` if you'd like it available everywhere:

```bash
sudo mv dist/stack-stitcher /usr/local/bin/
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

<!-- TODO: Document how the file is located — does it auto-detect ./compose.yml, or can the user pass a path/flag? Add that here. -->

### Key bindings

<!-- TODO: Fill in the real key bindings from your Bubble Tea Update() logic, e.g.:

| Key         | Action                        |
| ----------- | ----------------------------- |
| `↑` / `↓`   | Navigate services             |
| `/`         | Fuzzy search                  |
| `enter`     | Select / run action           |
| `y`         | Copy to clipboard             |
| `q`         | Quit                          |

-->

## Tech stack

- **Language:** Go
- **TUI:** Bubble Tea, Bubbles, Lip Gloss (Charm)
- **Compose parsing:** `compose-spec/compose-go`
- **Extras:** fuzzy matching (`sahilm/fuzzy`), clipboard (`atotto/clipboard`), shell parsing (`go-shellwords`)

## Project layout

```
.
├── main.go        # Entry point — starts the Bubble Tea program
├── src/
│   └── model/     # Bubble Tea model (state, Update, View)
├── Makefile       # dev / build targets
├── go.mod
└── go.sum
```

## Development

```bash
make dev     # run locally
make build   # compile to dist/stack-stitcher
```

Contributions, issues, and feature ideas are welcome.

## License

Released under the [MIT License](LICENSE). © 2026 Filipe Molina.
