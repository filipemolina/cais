# TODO

Working list for Stack Stitcher. Sources: the roadmap in `docs/ROADMAP.md`,
the guiding principles in `docs/DESIGN.md`, plus review findings.

Legend: **[P]** = from the original plan/roadmap, **[S]** = suggested next
step, **[H]** = housekeeping.

**This file is the flat list of what is left. `docs/ROADMAP.md` is the order to
do it in, and why** — it carries the decisions already taken with the owner, so
work resumed mid-sequence does not re-litigate them. Phases 0–9 of that roadmap
are done — every tab is live and the alpha roadmap is complete. What remains is
the post-alpha list. (See `docs/ROADMAP.md`.)

`README.md`, `docs/DESIGN.md`, `docs/ROADMAP.md`, and this file are the current
documentation.

---

## Remaining from the original plan

- [x] **[P] Edit existing services** — inline editing works: `e` on the
  Services details panel opens a `textarea` with the service's YAML
  fragment; `ctrl+s` saves through `utils.ApplyServiceFragment`, `ctrl+o`
  opens the same fragment in `$EDITOR`, and `esc` cancels (confirming first
  if the buffer changed). The editor owns the keyboard while open, so action
  keys are plain text. Live YAML syntax validation is shown on the status
  line; a save that fails compose validation keeps the editor open with the
  error and leaves the file untouched. **Remaining:** the deferred draft
  mechanism (resume a rejected edit from `$XDG_CACHE_HOME/stack-stitcher/drafts/`).

- [x] **[S] Blank lines are not preserved across writes** — accepted, not
  fixed. `yaml.v3` round-trips comments but not blank lines, so every write
  (group tags included, long before edit-services) closes up the spacing
  between services. Carrying them through as marker comments was built and
  then deliberately removed: a blank line inside a block scalar (`command: |`)
  is part of the string, so the trick needs to know where it must not apply,
  and silently rewriting the user's data is a worse failure than losing their
  spacing. Don't reintroduce it without a real YAML round-tripping library.

- [x] **[H] Flaky bootstrap tests** — fixed, and the diagnosis in this entry
  was wrong: not a timing problem in the rig but a real one in the app. Init's
  own `GetConfig` and each test's explicit message both reported
  `ErrNoComposeFile`, and `Update` answered each by assigning a *fresh*
  `CreateComposeFileModal` — so the second one landed mid-typing and reset the
  filename field. Twelve backspaces against a field refilled after the sixth
  leaves `compose.y`, which is why the failure read "must end in .yaml". The
  modal now opens only when nothing else owns the screen, which is the right
  behaviour anyway: a background reload has no business closing a modal the
  user is working in. Six consecutive full-suite runs under `-race` pass.

- [x] **[H] Panel keys aren't testable through the rig** — fixed: panel
  keybindings match with `key.Matches`, which compares `msg.String()` against
  the binding strings. `tea.KeyPressMsg.String()` returns `Text` when it is
  set, so a key sent to the rig with only `Code` matched nothing. The rig
  now has a `letterKey` helper that sets both `Code` and `Text`, and
  `TestRigGroupListEditKey` verifies 'e' reaches the focused groups list
  through a real `tea.Program`. Modal special keys (esc/enter/tab/backspace)
  still work with `keyPress` because Code alone resolves to the right
  string for those. Panel-key coverage is no longer blocked for Phase 3.

- [x] **[P] Compose Files page** — done across Phases 8–9. Phase 8 landed the
  minimum: the active file's path on the title row, a read-only scrollable
  view of its raw contents, and `E` to edit it in `$EDITOR`. Phase 9 completed
  it: a hand-rolled, line-oriented YAML highlighter colors keys, quoted
  strings and comments from the active theme (display-only — it changes no
  byte, so the view still matches the file `E` opens; it tracks block scalars
  so a `command: |` body is not colored as structure); and `b` opens a picker
  listing the YAML files in the active file's directory, with the loaded one
  marked, and Enter switches — which is exactly passing `--file` at runtime
  (AppModel points the source at the chosen path and reloads with `GetConfig`,
  so every downstream consumer follows without further work).

- [x] **[P] Settings page** — dropped as a page. The tab was a placeholder with
  two rows of content in it, and each of those settings has a better home:
  the compose file is a `--file` flag (per run, explicit), and the theme is a
  picker modal once colors are centralized. What persists goes to
  `~/.config/stack-stitcher/config.yaml` with no page to maintain.

- [x] **[P] About modal** — `a` opens a read-only About overlay carrying the
  reserved ASCII `LOGO` (`src/constants/Branding.go`), the wordmark and
  slogan, the version (`constants.Version()`), the license (MIT), and the repo
  link. It closes on the same three keys as the help overlay - `a` (the one
  that opened it), `esc`, and `q` (which closes the overlay, not the app,
  while it owns the keyboard). `a` is advertised in the `?` help overlay
  rather than the footer: the footer is width-constrained (its narrow-terminal
  wrapping is its own open item), and `?` is the comprehensive list, so the
  footer stays lean while `a` is still discoverable. `?` stays the help key;
  `a` does not collide with any panel or list binding.

- [x] **[P] Edit group membership** — `e` on the groups list reopens the
  `ServiceChecklistModal` pre-checked with the group's current members, and
  saving reconciles the diff through `utils.SetGroupMembers`: newly checked
  services are tagged, newly unchecked ones untagged, in a single
  read-modify-write pass (so there is no crash window with a half-applied
  edit). Unchecking every service removes the group, matching delete. Enter
  emits `cmds.EditGroupRequestMsg`; `AppModel` binds the loaded file and runs
  `cmds.EditGroup`. This was Phase 8's first half.

- [ ] **[P] Group rename** — `DESIGN.md` §3 lists it as unsupported. It's a
  straightforward `yaml.Node` walk (retag every service that carries the
  name); worth doing once membership editing exists.

- [x] **[P] `--file` / `--directory` flag** — `-f`/`--file` opens exactly the
  file named; `-d`/`--dir` resolves one inside that directory in Docker's
  order. `utils.ComposeSource` carries whichever was given from `main.go` to
  the resolver, and the resolved path reaches the footer, the YAML writers and
  every docker call. That last part had to land **first**, as its own change:
  until every `docker compose` invocation passed `--file`, the flag would have
  had the panel describing one file while the commands acted on another. The
  two flags are refused together, and a bad path fails before the alternate
  screen. See *Which compose file* in `docs/DESIGN.md`. This was Phase 7.

## Suggested next steps

- [x] **[S] Theme picker modal** — `T` (shift+t) opens a picker listing
  every registered theme (`stitcher-dark`, `stitcher-light`,
  `stitcher-ocean`, `stitcher-ember`), sorted by name. Cursor movement
  previews the theme live — the entire UI behind the modal repaints on
  each cursor step. `Enter` applies and persists to
  `~/.config/stack-stitcher/config.yaml` (or `$XDG_CONFIG_HOME`); `Esc`
  restores the theme that was active when the picker opened. The saved
  theme is loaded on startup in `main.go`, before the program starts. Two
  new themes (`stitcher-ocean`, `stitcher-ember`) join the existing pair
  to make the picker worth having. The config struct
  (`src/config/config.go`) is designed to absorb future fields (default
  file, keybinding overrides) without changing callers.

- [x] **[S] Periodic container refresh** — re-poll `docker compose ps`
  every five seconds while a compose project is loaded and no modal is open.
  Background results refresh status without clearing an unrelated action
  error; a recovered poll clears its own error banner.

- [x] **[S] Confirm destructive Remove** — `x` now opens the reusable
  `ConfirmModal` before it runs `docker compose rm -fs` for a service or
  group.

- [x] **[S] Drop the `jq` dependency** — `DockerComposePs` now invokes
  `docker compose ps --format json` directly. Go parses both JSON-array and
  legacy NDJSON output, so `jq` is no longer a runtime requirement.

- [x] **[S] Rename the module for distribution** — `go.mod` is now
  `module github.com/filipemolina/stack-stitcher`, so
  `go install github.com/filipemolina/stack-stitcher@latest` works. Version
  stamping and `--version` landed in Phase 7: `constants.Version()` prefers
  the `-ldflags -X` stamp, falls back to the commit from the build info, and
  the nav bar renders it dimmed beside the wordmark.

- [x] **[S] One keymap** — every binding now lives once in `src/keys`.
  Components match with `key.Matches`; `KeybindingBar` asks `keys.Active` which
  bindings are live instead of hand-listing them, so the footer can no longer
  advertise a key no handler implements. `TestFooterHints` pins all ten
  contexts. The duplicated key→action maps in both details panels are gone,
  replaced by one `dockerActionFor`. See *Where keybindings live* in
  `docs/DESIGN.md`.

- [x] **[S] The lists don't own their keymaps** — both lists now install
  `keys.ListKeyMap()` instead of `list.DefaultKeyMap()`, keeping only cursor
  movement, `g`/`G` and `/`, so `d` no longer pages the list while opening the
  delete confirm. Filtering is a supported mode and behaves as an overlay: the
  list reports `OwnsKeyboard()`, `AppModel` stands down from its own keys while
  it does, and the footer advertises the filter's keys instead of inert ones.
  `ctrl+c` became its own binding so it beats every claim on the keyboard —
  it did not quit while a modal was open before. See *The lists do not get to
  keep `list.DefaultKeyMap`* in `docs/DESIGN.md`.

- [x] **[S] Show the parsed compose file in the footer** — `AppModel`
  broadcasts the file it resolved as `cmds.SetComposeFileMsg`, and
  `KeybindingBar` renders it dimmed immediately left of the global keys,
  degrading full path → basename → dropped as the terminal narrows.
  Docker's file priority stays fixed and identical to Docker's on purpose
  (making it configurable would desync the panel from the `docker compose`
  calls, which pass no `-f`), so *saying which file won* was the fix, not a
  setting. See *Which compose file* in `docs/DESIGN.md`. When several
  candidates exist the footer marks the winner with `+N` and the `?` overlay
  lists the losers — `GetComposeFileName` returns every candidate in priority
  order.

- [x] **[S] The new global keys** — digits `1`–`3` jump to pages and `[`/`]`
  step them with wraparound (`alt`+letter kept as an alias: macOS Terminal.app
  and iTerm2 do not send Option as Alt by default, so the chords were silently
  dead for part of the audience). `enter` selects in both lists as an alias
  for `space`, the nav renders each tab's digit instead of underlining a
  letter, and `esc` is a real "back" (details → list). The three constraints
  from the keymap work held: `esc` clears an applied filter before it moves
  focus (`KeepsEsc`, with a focus-then-clear ladder when the filtered list is
  not focused), the digits live inside the `keyboardOwned()` guard so they
  stay letters while a filter is typed, and `tab` while filtering stays inert
  — making it apply-and-move would resurrect the one-key-two-jobs collision
  the list keymap work removed. Two labels may now share a first letter; the
  uniqueness guard went with the underline. See *Navigation and focus* in
  `docs/DESIGN.md`. The `?` overlay picked up the `alt` aliases and the
  brackets, which the footer had no room for.

- [x] **[S] `?` help overlay** — `?` opens `components.HelpOverlay` through
  `cmds.OpenHelpModal`, rendered from `keys.Catalog(ctx)`: every binding
  grouped by scope, with rows that do nothing on the screen it opened from
  dimmed. Availability comes from a snapshot (`AppModel.helpContext`: page,
  focus, selection, filter state via the lists' new `FilterState`); a modal
  freezes the panels, so the snapshot cannot go stale. Closes with
  `?`/`esc`/`q` — `q` closes only the overlay. It is the home for the
  `alt`+letter aliases (one derived `alt+g/s/f` row), the `[`/`]` brackets,
  `g`/`G`, `shift+tab`, `ctrl+c`, and the losing compose-file candidates the
  footer can only count. The footer's global group gained `? help`.

- [ ] **[S] The footer wraps on a narrow terminal** — predates the compose
  file segment (which drops itself rather than contributing to this). Below
  roughly 60 columns the context hints plus the global keys exceed the width,
  and the bar wraps to two or three lines, eating body rows. The bar needs to
  shed hints in priority order the way the file name already does. Same
  terminals show two other overflows worth fixing together: the group details
  table collides its column headers (`NAMEIMAGSTATHEALT…`) and the action
  buttons wrap into each other.

- [x] **[S] Centralize color into a `Theme`** — `appstyles.Theme`
  (`src/appstyles/Theme.go`) is one field per semantic token, built by a
  `newTheme` constructor that derives everything but a handful of base
  colors via `Lighten`/`Darken`, flipping which operator "raise" means by a
  `Dark` flag so the same deltas work in a light theme too.
  `appstyles.Themes` registers `stitcher-dark` (today's palette, byte-for-byte
  verified) and a new `stitcher-light`, closing the "unusable on a light
  terminal" risk; `appstyles.Active` is the one in effect, read fresh by every
  call site (`appstyles.Active.TextPrimary`, not a cached `var`) so a later
  switch actually repaints. Every legacy alias is gone, including three
  (`PrimaryColor`, `PrimaryFontColor`, `SecondaryFontColor`) the roadmap didn't
  name but turned out to be the three heaviest-used names in the codebase, and
  the five stray hexes are theme tokens now. Found and fixed a real bug along
  the way: a status pill's text color used `PanelBg`/`TextPrimary` as stand-ins
  for "dark"/"light", which only worked because the one theme that existed was
  dark; fixed with two theme-invariant fields, `InkOnLight`/`InkOnDark`, since
  a pill's own fill doesn't vary with the app's theme either. The
  background-bleed suites now run once per registered theme
  (`src/model/background_test.go`'s `forEachTheme`), verified against a
  deliberately broken throwaway theme to confirm the safety net actually
  catches something. See *Color lives on a Theme* and *Background tiers, and
  sealing them* in `docs/DESIGN.md`. This was Phase 6 in `docs/ROADMAP.md`.

- [x] **[S] CI + releases** — `.github/workflows/ci.yml` runs build, vet,
  gofmt and `go test -race` on push to main and on every PR; `release.yml`
  runs GoReleaser on a `v*` tag, building linux/darwin × amd64/arm64 with the
  version stamped in and drafting the release for review. `CONTRIBUTING.md`
  documents the loop. Verified with `goreleaser check` and a snapshot build.
  No Windows target: the app shells out to `docker compose` and hands the
  terminal to `$EDITOR`, and neither has been tried there.

- [ ] **[S] Expand test coverage via the e2e rig** — `src/model/rig_test.go`
  already drives the app in-process (used for the bootstrap flow). Extend
  to: create/delete group flow, docker actions against a fake `docker` on
  `PATH`, and the logs modal.

- [ ] **[S] Logs overlay improvements** — search/filter (`/`), line-wrap
  toggle, toggle timestamps, jump to top (`g`) / bottom (`G`).

- [x] **[S] Error banner lifecycle** — Esc now dismisses the banner. It is the
  next rung in esc's existing priority ladder (after a modal closes, a filter
  being typed owns the keyboard, and an applied filter keeps esc): when no
  stronger claim holds and a banner is showing, the first esc clears it and
  the next esc backs out of the details panel — the same one-key-one-job
  ladder a filtered list clears on. A recovered poll still clears its own
  banner, and other errors still clear on the next successful foreground
  operation; Esc is the manual dismissal that did not exist before. Auto-expire
  was not added — the "and/or" left it optional, and a fixed timeout risks
  expiring an error the user is still reading.

- [x] **[S] Re-record `demo/demo.gif`** — re-recorded against real containers:
  start a group, tail its logs, stop it, switch pages with a digit, start one
  service, open the `?` overlay. It is 3.9MB against the old 226KB, and mostly
  not because it is longer: since the theme work every frame paints the full
  screen instead of leaving most of it black, so frames no longer compress to
  nothing. The tape documents the `mpdecimate` + shared-palette pass that
  brings it down from 5.1MB — worth another look if it needs to be smaller
  still, most likely by cutting the logs section, which is the densest part.

- [x] **[S] Auto-select on navigation** — arrow keys in the list
  automatically select the item under the cursor, updating the details panel
  immediately. No separate "space to select" step. `space`/`enter` become
  aliases for "start the selected item". See `docs/PLAN-UX-IMPROVEMENTS.md`
  for implementation details.

- [x] **[S] `n` works on both panels** — the `n` key opens the create group
  modal from either the list or details panel on the Home page. This fixes
  the onboarding issue where the empty state says "press n" but the key only
  works on one panel. See `docs/PLAN-UX-IMPROVEMENTS.md`.

- [x] **[S] Action feedback with spinner** — show a spinning animation while
  docker actions (start, stop, restart, pull, remove) are in progress. The
  spinner appears in the title pill area (replacing the status pill) and in
  the action buttons area (replacing the buttons). Action keys are disabled
  while an action is pending. See `docs/PLAN-UX-IMPROVEMENTS.md`.

- [x] **[S] Error modals for foreground errors** — show foreground errors
  (from docker actions, config loads, etc.) in a modal dialog instead of the
  banner. Background poll errors keep the banner to avoid modal fatigue. See
  `docs/PLAN-UX-IMPROVEMENTS.md`.

- [x] **[S] Service details panel redesign** — redesigned the right-pane details
  for individual services to match the visual polish of the group details panel.
  Replaced the old `BasicInfo` card with: a service header (name, image, status
  dot with state/health/uptime), a compact two-column PROPERTY|VALUE config
  table (ports, restart policy, networks, volumes, depends_on, healthcheck,
  pull policy, PUID/PGID, memory limits, labels), and an improved runtime stats
  table (memory, CPU, network I/O, disk I/O, PIDs, uptime). Information was
  curated for the self-host enthusiast audience. All existing functionality
  (inline editor, docker actions, action buttons, logs modal) preserved.
  Branch: `service-details-redesign`.

## Housekeeping

- [x] **[H] Document the current build/install path** — README now correctly
  says that `make build` runs `go install .` and installs to
  `$(go env GOPATH)/bin`.

- [x] **[H] Update `demo/demo.tape`** — header, vocabulary and keys all
  brought up to date; it had been switching pages by tabbing onto the nav and
  pressing Right, which stopped working in Phase 4.

- [x] **[H] Empty `Name:` in the service details panel** — the `BasicInfo`
  card used `service.ContainerName` (the optional `container_name:` field,
  usually empty) for the `Name:` row, so it read blank while the list beside
  it showed the name. It now uses `service.Name` (the `services:` key, always
  set). PUID/PGID are optional env-var-derived fields (common in the *arr
  stack, absent for most services), so their row is dropped entirely when
  neither is set rather than rendering empty labels; when only one is set,
  only that one appears.

- [x] **[H] Delete `reference/*.go.bak`** — removed from disk. Bubble Tea
  tutorial leftovers; the `/reference/` directory was already gitignored, so
  this was a disk-only cleanup.

- [x] **[H] Fix `terminalWidht` typo** — renamed the `configModel` field to
  `terminalWidth` and its uses in `Update.go` / `View.go`.

- [x] **[H] Prune stray artifacts** — deleted `vhs-test.gif`/`vhs-test.tape`
  from the repo root. They were a one-off "echo hello world" VHS scratch
  test, not part of any demo.
