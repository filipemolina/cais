# Roadmap to the first alpha

This is the ordered plan the current work follows, and it is **live**.
`TODO.md` is the flat worklist; this file is the order and the reasoning, so
picking up mid-sequence does not mean re-deciding what was already settled.

The aim is not a finished product. It is a foundation someone else can extend
without re-litigating decisions, plus enough polish to publish a first alpha.

## Conventions

Each phase is a feature branch of small commits, merged with `--no-ff` so a
phase can be reverted as a unit. `go build ./... && go vet ./... && go test ./...`
green at **every** commit, not just at the merge. The merge commit's hash then
goes into the *Where we are* table as its `done` marker (a tiny "Pin the phase
N merge hash" commit straight to main — the hash cannot exist before the
merge does), and the phase branch is deleted.

Behaviour that only shows up on screen gets checked in the real app with VHS
(`vhs` is installed; write a tape with `Screenshot "name.png"` and run it from a
scratch directory — note that VHS needs its paths quoted, and sometimes drops
the last screenshot, so re-run if the file is missing). For a faster loop, a
throwaway `go run` program that renders one component through
`ansi.Strip(m.View().Content)` catches layout and styling mistakes before they
are committed; phases 4 and 5 used it for the nav and the help overlay.

Docs are part of the phase, not a follow-up: `README.md` for what a user needs to
know, `docs/DESIGN.md` for why a decision went the way it did, `TODO.md` for
what is left. When a phase lands, its section here is removed and the design
moves to `docs/DESIGN.md` — see the *Where we are* table note above the phase
sections.

## Decisions already taken with the owner

Do not re-open these without asking:

- **Tabs for the alpha are Groups, Services, Files.** No dead placeholder tabs.
- **Pages switch with digits `1`–`3` and `[`/`]`,** keeping `alt`+letter as an alias.
- **No statistics page.** Resource numbers belong as columns in the tables that
  already exist, not on a page of their own. `docker stats` is slow, it needs its
  own polling design, and ctop/lazydocker already own that niche.
- **No prefix key** (tmux/zellij style). The reasoning is written up in
  `docs/DESIGN.md` — prefixes exist to address a host that is hosting another
  program that owns the keyboard, and this app has no guest.
- **The compose file priority order stays fixed and identical to Docker's.**
  Reasoning in `docs/DESIGN.md` under *Which compose file*.
- **Colors get centralized into a `Theme` value** before the alpha, so a theme
  picker modal can land right after it.

## Where we are

| Phase | Status |
| --- | --- |
| 0 — Rename the module, rename the Dashboard page, drop the Settings tab | done (`231adc7`) |
| 1 — One keymap in `src/keys` | done (`9a68171`) |
| 2 — Footer shows the parsed compose file | done (`62416ef`) |
| 3 — The lists own their keymaps | done (`55173d0`) |
| 4 — The new global keys | done (`a64ec73`) |
| 5 — `?` help overlay | done (`37bf74a`) |
| 6 — Centralize color into a `Theme` | done (`88ce42d`) |
| 7 — Release plumbing | done (`01f75e9`) |
| 8 — Edit group membership, then the Files page | done (`ffe9fed`) |
| 9 — Complete the Files page: syntax highlighting, browse and switch | done (`f450a55`) |

Phases 0–9 are described in `docs/DESIGN.md` (*Where keybindings live*, *Which
compose file* — now including *One resolution, passed down* and the two flags —
*The lists do not get to keep `list.DefaultKeyMap`*, *Navigation and focus*,
*Color lives on a Theme*, *Background tiers, and sealing them*, *Saying which
build this is*, *Editing group membership*, *The Files page*) rather than here,
because they are now how the app works rather than a plan.

That is the whole roadmap to the alpha: every tab is live, and the first
group-membership wall is gone. What follows is the post-alpha list.

## Explicitly post-alpha

The theme picker is done (`T` opens it, cursor previews live, Enter persists
to `~/.config/stack-stitcher/config.yaml`). The config file exists and
already stores the theme; default file and keybinding overrides are the
remaining fields the keymap struct makes a load-and-merge. Additional
themes beyond the 14 shipped. Live CPU/MEM columns from `docker
stats`. An `x`-style action menu.

**UX improvements** are done: auto-select on navigation, `n` on both
panels, action feedback with spinner, and error modals for foreground
errors (see `docs/PLAN-UX-IMPROVEMENTS.md`).

The **service details panel** was redesigned to match the visual polish
of the group details panel: a service header with status dot, a compact
two-column configuration table curated for self-host enthusiasts, and an
improved runtime stats table. All existing functionality (inline editor,
docker actions, logs modal) preserved. See `docs/DESIGN.md` §Services
layout.

## Loose ends worth knowing about

- **`CPU: 0.0` in the services list is not telemetry.**
  `src/apptypes/ServiceListItem.go` renders `service.CPUPercent`, which is the
  *configured* `cpus:` limit from compose-go — 0.0 for nearly every file, but it
  reads as live usage. Drop or relabel it until real stats land.
- **The footer wraps below roughly 60 columns**, and so do the details table
  headers and the action buttons. Tracked in `TODO.md`; the bar needs to shed
  hints in priority order the way the compose file name already does.
- **Panel keypresses through the e2e rig — fixed.** `TODO.md` has the details:
  panel keys need `Text` set on the `tea.KeyPressMsg` because `key.Matches`
  compares `msg.String()` against the binding strings. `TestRigGroupListEditKey`
  now verifies a panel key ('e') reaches the focused groups list through the
  full program. The rig gained a `letterKey` helper for printable keys; special
  modal keys (esc/enter/tab/backspace) still go through the existing `keyPress`
  helper.
- **The two flaky bootstrap tests were an app bug, not a rig bug** (fixed in
  Phase 7). Worth remembering before blaming the rig again: a failed reload was
  replacing the open modal with a fresh one. If a rig test starts failing
  intermittently, ask what else the app is doing to itself first.
- **`--version` is unstamped in a `go run` / plain `go build`.** It reports the
  commit instead, which is intended. Only `make build` and the release build
  stamp a version, so a screenshot of a dev build shows a hash in the nav bar.
