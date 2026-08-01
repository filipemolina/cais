# TODO

Working list for Stack Stitcher. Sources: the roadmap in `docs/ROADMAP.md`,
the guiding principles in `docs/DESIGN.md`, plus review findings.

Legend: **[P]** = from the original plan/roadmap, **[S]** = suggested next
step, **[H]** = housekeeping.

**This file is the flat list of what is left. `docs/ROADMAP.md` is the order to
do it in, and why** — it carries the decisions already taken with the owner, so
work resumed mid-sequence does not re-litigate them. Phases 0–9 of that roadmap
are done — every tab is live and the alpha roadmap is complete. The post-alpha
sequence, which runs through the plans in `docs/plans/` and ends at
`launch-and-outreach.md`, is in `docs/ROADMAP.md` §*The order after the alpha*.

`README.md`, `docs/DESIGN.md`, `docs/ROADMAP.md`, `docs/plans/`, and this file
are the current documentation.

Two open items below are launch gates rather than nice-to-haves, and are
called out as such in the roadmap: **write safety** (new, see below) and the
**footer wrap on a narrow terminal**, which is visible in the demo recording.

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

- [x] **[P] Group rename** — `R` on the groups list opens a rename prompt
  pre-filled with the group's current name; Enter writes a single
  read-modify-write pass that retags every service carrying the old name
  (`utils.RenameGroupTag`). The unchanged name and name collisions are
  refused inline; a successful rename keeps the group selected through the
  reload. Scoped to the loaded compose file, like every other group op —
  tags in other files of a multi-file project keep the old name (see
  `docs/DESIGN.md` §3).

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

- [x] **[S] `?` help overlay** — `?` opens `helpoverlay.New` through
  `cmds.OpenHelpModal`, rendered from `keys.Catalog(ctx)`: every binding
  grouped by scope, with rows that do nothing on the screen it opened from
  dimmed. Availability comes from a snapshot (`AppModel.helpContext`: page,
  focus, selection, filter state via the lists' new `FilterState`); a modal
  freezes the panels, so the snapshot cannot go stale. Closes with
  `?`/`esc`/`q` — `q` closes only the overlay. It is the home for the
  `alt`+letter aliases (one derived `alt+g/s/f` row), the `[`/`]` brackets,
  `g`/`G`, `shift+tab`, `ctrl+c`, and the losing compose-file candidates the
  footer can only count. The footer's global group gained `? help`.

- [x] **[S] The footer wraps on a narrow terminal** — fixed, along with the two
  other overflows the same terminals showed. All three were one mistake:
  lipgloss pads to `Width` but does not truncate, so a fixed set of controls
  squeezed past its own labels wraps on the cell rather than giving anything up.

  **"Below roughly 60 columns" was wrong — it is closer to 130.** Measured
  while recording the README screenshots (2026-07-31): at 1280px / 16pt
  (~133 columns) the Groups footer pushed `q quit` onto a second line; at
  1440px (~150 columns) it fit. The worst context is the Services details panel
  behind the logs overlay, which advertises ten hints. It was visible in
  `demo/demo.gif`, which made it the first bug a visitor saw, and
  `docs/plans/launch-and-outreach.md` listed it as a launch gate for that
  reason.

  The bar now sheds whole hints in the order `keys.Priority` declares, and the
  group table drops whole columns in the order `dropOrder` declares. Both
  orders are deliberately not their display orders, and both keep the thing
  that makes shedding safe rather than merely lossy: `? help` on the bar (it
  opens the overlay everything shed went to), the name column in the table (a
  row with no identity says nothing). Fixed with them: table cells truncated to
  the full column ran flush into the next value, and the group's summary line
  wrapped and cost the table a row.

  The pattern started with the details panels' action row, which shed whole
  buttons before it was removed for being an unclickable chip (see *The panel
  footer* in `docs/DESIGN.md`); `git show 63ea952^:src/components/chrome/PanelFrame.go`
  is where that worked example now lives. Three surfaces have wanted the same
  fix, so it is worth reaching for the fourth time before reaching for
  `MaxHeight` alone: clipping keeps the layout intact but says nothing about
  what was lost.

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

- [ ] **[S] Write safety for the compose file** — a launch gate, from
  `docs/plans/launch-and-outreach.md`. The app rewrites the user's compose
  file in place and the only safety net is that an invalid write is refused.
  There is no backup, no undo, and the one thing a write does not preserve
  (blank lines between services) is documented in this file rather than
  anywhere the user will see it. Minimum: write a `.bak` beside the file on
  the first write of a session, or notice the file is not in a git repo and
  say so once. Nobody lets an unfamiliar tool rewrite the file their homelab
  runs on without one of those. It now gates a second thing:
  `docs/plans/resources-page.md` Phase 2 adds a write surface and waits behind
  this.

- [ ] **[H] `ReadConfigFile` invents the project name** — it passes
  `cli.WithName("stack-stitcher")`, which sits at the top of compose's name
  resolution ladder and therefore *overrides the file's own `name:` key*.
  Measured against `demo/fixtures/compose.yaml` (which declares
  `name: homelab`): the app resolves `stack-stitcher` and computes
  `stack-stitcher_navidrome-data`, while the `docker compose` calls the app
  itself makes resolve `homelab` and create `homelab_navidrome-data`. Removing
  the option yields `homelab` for that file and `mocks` (the directory
  basename) for one with no `name:` — exactly docker's own rules.

  Harmless today, because nothing reads `project.Name`. A silent wrong answer
  the moment anything correlates the file against the daemon, which is
  `docs/plans/resources-page.md`'s whole job — so it is that plan's Phase 0.
  Check one thing before deleting the line: a directory whose basename is not
  a legal project name (spaces, uppercase) may be why it was there.

The six plans below came out of the 2026-08-01 feature round. Each is written
up in full in `docs/plans/`; `docs/ROADMAP.md` has the order and the reasons.

- [ ] **[S] Group table legibility** (`docs/plans/group-table-legibility.md`) —
  PORTS renders `0.0.0.0:6881->…` for every row and IMAGE renders
  `lscr.io/linuxse…` for three different services. Both columns get a
  formatter: published host ports from the file, and an image reference that
  sheds registry → namespace → tag instead of truncating the name off the end.
  A day, pure functions, and both defects are in `demo/screenshot-groups.png`.

- [ ] **[S] Docker preflight** (`docs/plans/docker-preflight.md`) — five states
  (no binary, no compose plugin, daemon down, socket permissions, a
  `DOCKER_HOST` pointing elsewhere), told apart by which probe failed rather
  than by parsing error text, each with the exact command that fixes it on this
  distro. **Decided: the app never installs, starts or configures anything** —
  the reasoning is in the plan so it does not get re-argued.

- [ ] **[S] Service URLs** (`docs/plans/service-urls.md`) — a `Web` row in the
  service details panel carrying a real OSC 8 hyperlink (verified zero-width to
  lipgloss, so it costs no layout), with the host taken from `SSH_CONNECTION`'s
  server field — the address the client demonstrably just used — and `y` to
  copy via OSC 52, which works over SSH. The app never spawns a browser.

- [ ] **[S] Usage overlay** (`docs/plans/docker-disk-usage.md`) — `u` opens
  disk and memory as horizontal bars. Not a page, so the *no statistics page*
  decision stands; fetched on open with a spinner because `docker system df` is
  2.3 s. On the author's machine it reports 42 GB of reclaimable images that no
  tool in the stack currently mentions.

- [ ] **[S] Resources page** (`docs/plans/resources-page.md`) — networks and
  volumes as a fourth tab, read-only. The feature is the four states a resource
  can be in; **created-but-undefined is a data-loss detector** (rename a volume
  in the file and `up` gives you a fresh empty one while the old keeps the
  data). Phase 2 (create/attach) waits for write safety; deletion needs a typed
  confirmation, not `y`/`n`, and is not planned.

- [ ] **[S] Adopt unmanaged containers**
  (`docs/plans/adopt-unmanaged-containers.md`) — three categories with three
  offers: this project's orphans, another project's containers (whose
  `config_files` label makes "switch to that file" free), and `docker run`
  leftovers. Adoption generates a service block by diffing `docker inspect`
  against the *image's* config — 4 real environment variables out of 10 — and
  opens it in the inline editor. Never a silent write; the container is never
  touched. Phase 2 needs `image-search.md`'s insert primitive.

- [ ] **[H] `TestRigRenameGroup` is flaky** — fails roughly one run in three,
  on `main`, independent of any current branch (four `-count=1` runs on
  2026-07-31: two passes, two failures). Same family as the bootstrap-test
  flakiness that turned out to be a real app bug rather than a rig timing
  problem, so start by asking what else the app is doing to itself during a
  rename before adding a wait.

- [ ] **[S] Expand test coverage via the e2e rig** — `src/model/rig_test.go`
  already drives the app in-process (used for the bootstrap flow). Extend
  to: create/delete group flow, docker actions against a fake `docker` on
  `PATH`, and the logs modal.

- [ ] **[S] Logs overlay improvements** — search/filter (`/`), line-wrap
  toggle, toggle timestamps, jump to top (`g`) / bottom (`G`).

- [ ] **[M] Mouse interaction** — deferred to a later version. Bubbletea reports
  clicks (`tea.MouseClickMsg`) once mouse mode is enabled; the work is deciding
  which surfaces answer them (list rows, panel focus, the modal buttons) and
  keeping every one of them reachable from the keyboard, which is still the
  primary interface.

  This is why the details panels' action button row was removed: a padded,
  filled chip reads as a clickable control, and clicking it did nothing. See
  *The panel footer* in `docs/DESIGN.md` for what the row was and what it cost
  to keep. When mouse support lands, restoring it is a reasonable first
  consumer — `git show 63ea952^` has the whole thing, including the shed-in-
  priority-order layout and its tests. Until then, if the panels need a local
  statement of their own verbs, it should be a plain key-hint line rather than
  chips: a keyboard-only affordance should look like text.

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

- [x] **[S] Re-record `demo/demo.gif`** — done twice. The first pass recorded
  against real containers and landed at 3.9MB (up from 226KB, and mostly not
  because it was longer: since the theme work every frame paints the full
  screen instead of leaving most of it black, so frames no longer compress to
  nothing).

  Re-done on 2026-07-31 against the new homelab fixture, and the size problem
  is solved: **2.4MB, from a 12MB raw recording**, while being longer than
  either predecessor. The palette pass alone was not enough; what worked is
  three filters together — `mpdecimate`, `fps=10`, and rendering at 900px wide,
  which is roughly what GitHub displays a README image at anyway. Recording
  large and scaling down beats recording small, because the footer wrap above
  gets worse the narrower the terminal is. `demo/screenshots.tape` is the
  companion that produces the five stills; both tapes export a throwaway
  `XDG_CONFIG_HOME` so a recording cannot overwrite the recorder's own theme.

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

- [x] **[H] The HEALTH column was never populated** — `docker compose ps
  --format json` emits the key `Health`; `apptypes.DockerContainer` called the
  field `HealthStatus` with no json tag, so `encoding/json` never matched it.
  Every container read `-` in the group member table and in the service status
  line, healthy ones included. Fixed with a `json:"Health"` tag.

  **The reason it survived this long is the part worth keeping:** the parser
  tests passed. Their fixtures used `"HealthStatus"` — the Go field name, which
  no Docker ever sends — so the test agreed with the code and both were wrong
  about reality. The fixtures now carry the real key and assert the parsed
  value. A fixture invented from the struct instead of captured from the tool
  tests nothing but itself.

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

- [x] **[H] Foreground errors vanished behind an open modal** — every
  foreground error path guarded its modal on `activeModal == nil` with no
  `else`, so an error arriving while any modal was up was dropped: no modal,
  no banner. Pressing `s` and then `?` before the action came back reported
  nothing at all. The guard is right — a modal the user opened deliberately
  is not something a late error gets to close — so the fix is a fallback,
  not a removal: `AppModel.reportForegroundError` takes the modal when the
  screen is free and the banner when it is not, and the eight sites that
  repeated the guard call it. Note the asymmetry, which is load-bearing:
  the banner path sets `lastErrorFromPoll = false` (a foreground error owns
  the banner, so a later successful poll must not clear it), and the modal
  path deliberately does not (it leaves the banner untouched, so a poll
  error still showing there is still the poll's to clear). Getting that
  backwards strands the poll error, because a recovered poll only clears
  what it put up itself.

- [x] **[H] `formatMemUsage` duplicated, and applied twice** — the function
  existed verbatim in `components` and `apptypes`, and the value went
  through it twice: `containerMemUsage` formatted into the list item's
  `MemUsage`, then `ServiceListItem.Description` formatted the result again.
  It was invisible only because nothing ever assigned
  `ServiceListItem.MemPerc`, so the second pass had no percent to append —
  set that field and the row reads `21.71MiB (0.07%) (0.07%)`, because the
  second pass finds no `/` left to split on. The item now carries docker's
  raw strings (`containerMem` returns the pair unformatted and `MemPerc` is
  populated), `apptypes.FormatMemUsage` is the single copy, and formatting
  happens once at render time. **The rule to keep:** the function is not
  idempotent, so store raw and format at the edge. A test asserts the
  non-idempotency so reintroducing format-on-the-way-in fails loudly.

- [x] **[H] Runtime stats table hid partial data** — `renderRuntimeStats`
  bailed out unless memory, CPU, network or disk I/O was set: four of the
  six rows it can draw. A container reporting only PIDs, or only the uptime
  that comes from `docker compose ps` rather than `docker stats`, rendered
  nothing. The guard was redundant as well as incomplete — every row is
  already conditional and the `len(rows) == 0` check at the end asks the
  same question from what actually rendered — so it was dropped rather than
  extended, which fixes the omission by construction: there is no second
  list of fields left to fall out of step with the rows. One visible
  consequence, accepted: with `docker stats` unavailable a running service
  now shows a stats table holding just Uptime instead of no table at all.
