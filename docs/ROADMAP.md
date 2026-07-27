# Roadmap to the first alpha

This is the ordered plan the current work follows, and it is **live** — unlike
the dated specs and plans under `docs/superpowers/`, which are finished records.
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
| 8 — Edit group membership, then the Files page | **next** |

Phases 0–7 are described in `docs/DESIGN.md` (*Where keybindings live*, *Which
compose file* — now including *One resolution, passed down* and the two flags —
*The lists do not get to keep `list.DefaultKeyMap`*, *Navigation and focus*,
*Color lives on a Theme*, *Background tiers, and sealing them*, *Saying which
build this is*) rather than here, because they are now how the app works rather
than a plan.

## Phase 8 — Close the functional hole, then Files

- **Edit group membership.** `e` on the groups list reopens
  `ServiceChecklistModal` pre-checked with the group's current members; saving
  applies the diff by reusing the YAML walks in `cmds/CreateGroup.go` (tag) and
  `cmds/DeleteGroup.go` (untag). Today membership can only be set at creation,
  which is the first wall a real user hits. `e` = "edit the selected thing"
  matches `e` on the service details panel. Note that both commands now take
  the file name from `AppModel` rather than resolving it themselves (Phase 7),
  so whatever saves the diff must be given it the same way — the modal emits
  `cmds.CreateGroupRequestMsg` and `AppModel` supplies the file.
- **Files page, minimally.** Active file path, a read-only viewport of its
  contents, `E` to edit. Replaces the `PlaceholderPanel` so the alpha has no dead
  tabs. `-d`/`--dir` (Phase 7) makes the path worth showing in full: it is no
  longer always a bare name in the working directory.

## Explicitly post-alpha

Theme picker modal (a key opens the list of registered themes, cursor movement
previews live, `enter` applies and persists) and additional themes — Phase 6 is
what makes this small. A config file
(`~/.config/stack-stitcher/config.yaml`: theme, default file, keybinding
overrides — the keymap struct makes overrides a load-and-merge). Live CPU/MEM
columns from `docker stats`. Group rename. An `x`-style action menu. About modal.

## Loose ends worth knowing about

- **`CPU: 0.0` in the services list is not telemetry.**
  `src/apptypes/ServiceListItem.go` renders `service.CPUPercent`, which is the
  *configured* `cpus:` limit from compose-go — 0.0 for nearly every file, but it
  reads as live usage. Drop or relabel it until real stats land.
- **The footer wraps below roughly 60 columns**, and so do the details table
  headers and the action buttons. Tracked in `TODO.md`; the bar needs to shed
  hints in priority order the way the compose file name already does.
- **Panel keypresses through the e2e rig.** `TODO.md` has the details: rig tests
  that send a key target modals, which `AppModel.Update` handles on an early
  return. Phase 3 tested panel keys at the component and model level instead,
  which worked well enough that the rig gap is no longer blocking — but the rig
  is still the only place to test a full flow end to end.
- **The two flaky bootstrap tests were an app bug, not a rig bug** (fixed in
  Phase 7). Worth remembering before blaming the rig again: a failed reload was
  replacing the open modal with a fresh one. If a rig test starts failing
  intermittently, ask what else the app is doing to itself first.
- **`--version` is unstamped in a `go run` / plain `go build`.** It reports the
  commit instead, which is intended. Only `make build` and the release build
  stamp a version, so a screenshot of a dev build shows a hash in the nav bar.
