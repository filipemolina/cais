# Contributing

Thanks for looking. Issues, ideas and pull requests are all welcome — the app
is young enough that a good argument still changes its direction.

## Before you write code

Read [docs/DESIGN.md](docs/DESIGN.md). It records *why* things are the way they
are: the group-vs-profile vocabulary, why there is no prefix key, why the
compose file resolution order is fixed, where keybindings live. Most of it
exists because the alternative was tried first. [docs/ROADMAP.md](docs/ROADMAP.md)
is the ordered plan to a first alpha and lists the decisions already taken with
the owner; [TODO.md](TODO.md) is the flat worklist.

If you want to work on something sizeable, open an issue first. That is not
process for its own sake — the roadmap has an order to it, and some entries
depend on earlier ones landing.

## The layout

```
main.go              # entry point — flags, config load, starts the Bubble Tea program
src/
├── model/           # the top-level Bubble Tea model (AppModel, Init, Update, View)
├── components/      # one nested model per panel, list, modal and overlay
├── cmds/            # message types, and the tea.Cmds that produce them
├── apptypes/        # shared data types (list items, docker container, pages)
├── keys/            # every keybinding, declared once — panels, footer and ? all read it
├── utils/           # the non-Bubble Tea half: compose loading, YAML writing, docker exec
├── appstyles/       # the Theme type and the 14 registered themes
├── config/          # persisted preferences (~/.config/cais/config.yaml)
├── highlight/       # read-only YAML highlighting for the Files page
└── constants/       # layout widths, branding, focusable component list
demo/                # VHS tapes, the recorded gif and screenshots, and their fixture stack
docs/                # DESIGN.md (why), ROADMAP.md (order), plans/ (what's next, in sequence)
```

Two seams are worth knowing before you touch anything: `src/keys` is the single
declaration of every binding, and `appstyles.Active` is the single source of
color — read it fresh at each call site (`appstyles.Active.TextPrimary`, never a
cached package-level `var`) or a theme switch will not repaint what you wrote.

## The loop

```bash
make dev           # run against the compose file in the current directory
go build ./...
go vet ./...
go test ./...
gofmt -l .         # must print nothing
```

CI runs exactly these on every pull request, with `go test -race`. Run the race
detector locally too if you touch anything that shells out or streams: the
docker calls and the log stream each run on their own goroutine.

Keep every commit green, not just the branch tip. A commit that does not build
is a commit nobody can bisect through.

## Testing

Most behaviour is testable without a terminal:

- **Components** take messages and hand back a model — drive one directly and
  assert on the result (`src/components/serviceslist/Model_test.go`).
- **Rendering** is a string. `ansi.Strip(m.View().Content)` gives you the plain
  text of a component, which is enough to catch layout and styling mistakes
  (`src/components/mainmenu/Model_test.go`).
- **Whole flows** go through the e2e rig in `src/model/rig_test.go`, which runs
  a real `tea.Program` against an in-memory buffer.

Prefer the first two. The rig is the only way to test a full flow end to end,
but it is timing-based, so its assertions have to wait for output
(`r.WaitFor`) rather than sleep and hope.

Anything that shows up only on screen is worth checking in the real app with
[VHS](https://github.com/charmbracelet/vhs) before it is committed: write a
tape with `Screenshot "name.png"` and run it from a scratch directory. VHS
wants its paths quoted, and sometimes drops the last screenshot — re-run if the
file is missing.

## Style

Match the code around you. Two things that are not obvious from a diff:

- **Comments say why, not what.** The code already says what. A comment
  earns its place by recording a decision, a constraint, or a trap — ideally
  the thing that would otherwise be "fixed" back into a bug six months later.
- **A keybinding is declared once,** in [`src/keys/Keys.go`](src/keys/Keys.go).
  The panels match against it and the footer and `?` overlay render from it, so
  a key added anywhere else will not be advertised and may collide.

Commit messages: a short summary line, then prose explaining why. Look at
`git log` for the register.

## Releases

Maintainer-only, and automatic: pushing a `v*` tag runs GoReleaser, which
builds for linux and darwin (amd64 and arm64), stamps the version into the
binary, and opens a draft release to be reviewed before publishing.

```bash
git tag -a v0.1.0 -m "v0.1.0"
git push origin v0.1.0
```

`cais --version` reports the stamp. An unstamped local build reports
its commit instead, which is what a bug report wants anyway.
