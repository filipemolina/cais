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
green at **every** commit, not just at the merge. Behaviour that only shows up on
screen gets checked in the real app with VHS (`vhs` is installed; write a tape
with `Screenshot "name.png"` and run it from a scratch directory — note that VHS
needs its paths quoted, and sometimes drops the last screenshot, so re-run if the
file is missing).

Docs are part of the phase, not a follow-up: `README.md` for what a user needs to
know, `docs/DESIGN.md` for why a decision went the way it did, `TODO.md` for
what is left.

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
| 4 — The new global keys | done |
| 5 — `?` help overlay | **next** |
| 6 — Centralize color into a `Theme` | |
| 7 — Release plumbing | |
| 8 — Edit group membership, then the Files page | |

Phases 0–4 are described in `docs/DESIGN.md` (*Where keybindings live*, *Which
compose file*, *The lists do not get to keep `list.DefaultKeyMap`*, *Navigation
and focus*) rather than here, because they are now how the app works rather than
a plan.

## Phase 5 — `?` help overlay

`src/components/HelpOverlay.go`, opened by `?` through a `cmds.OpenHelpModal`
message handled in `AppModel.Update`, closed with `?` / `esc` / `q`. Rendered
**from `src/keys`** — grouped by scope, with bindings that are not currently
available dimmed — so it cannot drift from the handlers. Reuse `modalSurface`
(`src/components/PanelFrame.go`) and `renderKeyHints`. Add `? help` to the
footer's global group.

Two things are waiting on this overlay:

- The `alt`+letter aliases and the `[`/`]` brackets, which the footer has no
  room for.
- **The other compose-file candidates.** When more than one candidate name
  exists in the directory, the footer should mark it (`compose.yaml +2`) and the
  overlay should list the rest. `utils.GetComposeFileName` returns only the
  winner today and discards the others, so this needs a signature change —
  that is the whole of what is left of Phase 2.

## Phase 6 — Centralize color into a `Theme`

`src/appstyles/styles.go` holds good semantic tokens, but they are package-level
`var`s computed at init (`Lighten`/`Darken` of base colors) plus a block of
legacy aliases. Five other files carry stray hexes: `#B33A3A` in
`GroupNameModal.go`, `CreateComposeFileModal.go` and `model/View.go` (a *fourth*
red that is not `StatusError`), `#FAFAFA` in `LogsModal.go` and `View.go`,
`#3F3F3F` in `ContainersList.go`.

- Define `type Theme struct` with one field per semantic token and a constructor
  that derives the tiers (`BackgroundContent`/`Panel`/`Elevated`/`Recessed`,
  `BorderCard`, …) from a handful of base colors — the derivation rules move from
  package init into the constructor.
- A registry `map[string]Theme` with `stitcher-dark` (today's palette) as the
  default, plus `stitcher-light`, which closes the "unusable on a light terminal"
  risk. Drop the hardcoded `var lightDark = lipgloss.LightDark(false)`
  (`src/appstyles/styles.go:67`).
- **The real work:** the package-level style `var`s (`NormalTitle`, `DocStyle`,
  the `Selected*` family) are built at init and therefore freeze one palette.
  They become functions or methods on the active theme, so a later switch
  actually repaints. Then replace each stray hex above with a token.
- Retire the legacy aliases (`PaneColor`, `PanelBackgroundColor`,
  `SelectedPaneColor`, `FocusedPaneColor`, `BackgroundColor`) as call sites move.
- Free property: `src/appstyles/Background_test.go` and
  `src/model/background_test.go` already assert every tier is sealed.
  Parameterize them over every registered theme and a theme that leaves an
  unpainted cell fails CI.

## Phase 7 — Release plumbing

- A version variable stamped with `-ldflags -X`, plus `--version`.
- **First**, thread the resolved compose path into `utils.DockerCompose`,
  `utils.DockerComposePs` and `utils.DockerLogs` as `--file`; **then** add
  `-f`/`--file` and `-d`/`--dir` to `main.go` → `utils.GetComposeFileName` →
  `cmds.GetConfig` → `AppModel`. Without the first step the flag desyncs the UI
  from the commands it runs: the panel would describe one file while
  `docker compose start` acted on another. This ordering is not optional, and
  `docs/DESIGN.md` records why.
- GitHub Actions: `go build ./... && go vet ./... && go test ./...` on push/PR.
  GoReleaser for tagged binaries. `CONTRIBUTING.md`. Re-record `demo/demo.gif`
  (`demo/demo.tape` still says "profile" and `dist/stack-stitcher`).

## Phase 8 — Close the functional hole, then Files

- **Edit group membership.** `e` on the groups list reopens
  `ServiceChecklistModal` pre-checked with the group's current members; saving
  applies the diff by reusing the YAML walks in `cmds/CreateGroup.go` (tag) and
  `cmds/DeleteGroup.go` (untag). Today membership can only be set at creation,
  which is the first wall a real user hits. `e` = "edit the selected thing"
  matches `e` on the service details panel.
- **Files page, minimally.** Active file path, a read-only viewport of its
  contents, `E` to edit. Replaces the `PlaceholderPanel` so the alpha has no dead
  tabs.

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
