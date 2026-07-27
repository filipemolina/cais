# TODO

Working list for Stack Stitcher. Sources: the roadmap in `README.md`, the
guiding principles in `docs/DESIGN.md`, the "Out of scope / follow-ups"
sections of the specs in `docs/superpowers/specs/`, plus review findings.

Legend: **[P]** = from the original plan/roadmap, **[S]** = suggested next
step, **[H]** = housekeeping.

**This file is the flat list of what is left. `docs/ROADMAP.md` is the order to
do it in, and why** — it carries the decisions already taken with the owner, so
work resumed mid-sequence does not re-litigate them. Phases 0–4 of that roadmap
are done; **Phase 5, the `?` help overlay, is next.**

`README.md`, `docs/DESIGN.md`, `docs/ROADMAP.md`, and this file are the current
documentation. The dated specs and plans under `docs/superpowers/` are completed
historical records, not a live backlog.

---

## Remaining from the original plan

- [~] **[P] Edit existing services** — `$EDITOR`-based editing works: `e`
  opens one service's YAML, `E` opens the whole compose file. Writes are
  validated against the compose loader and refused (leaving the file
  untouched) rather than half-applied. The user edits real YAML, not a
  form, so every compose field is reachable. **Remaining:** Phase 3, editing
  inline in the details panel with a `textarea`, plus the deferred draft
  mechanism. See the live
  [design](docs/superpowers/specs/2026-07-25-edit-services-design.md) and
  [plan](docs/superpowers/plans/2026-07-25-edit-services.md).

- [x] **[S] Blank lines are not preserved across writes** — accepted, not
  fixed. `yaml.v3` round-trips comments but not blank lines, so every write
  (group tags included, long before edit-services) closes up the spacing
  between services. Carrying them through as marker comments was built and
  then deliberately removed: a blank line inside a block scalar (`command: |`)
  is part of the string, so the trick needs to know where it must not apply,
  and silently rewriting the user's data is a worse failure than losing their
  spacing. Don't reintroduce it without a real YAML round-tripping library.

- [ ] **[H] Flaky bootstrap tests** — `TestBootstrapModal_SkipServiceWritesEmptyFile`
  and `TestBootstrapModal_EmptyFilenameShowsInlineError` both fail perhaps one
  run in ten when the whole `src/model` suite runs, and pass alone (20/20).
  Reproduces on `main`, so it predates the edit-services work.
  The rig tests are timing-based; both end in a fixed `time.Sleep` followed by
  an assertion on `r.Latest()`, which reads only the bytes rendered since the
  previous call — so a frame arriving late lands in the wrong window. They want
  `r.WaitFor` instead. Worth fixing together with making panel keypresses
  testable through the rig, below.

- [ ] **[H] Panel keys aren't testable through the rig** — every rig test
  that sends a key targets a modal, which `AppModel.Update` handles on an
  early-return path that never reaches a panel. Keys sent to a *panel*
  through the rig did nothing when tried, so Phase 1/2 keypress coverage sits
  at the model level instead. Worth closing before Phase 3, which is entirely
  panel keys.

- [ ] **[P] Compose Files page** — currently a `PlaceholderPanel`. The tab
  label is already "Files". Minimum useful version: show which compose file
  is loaded and a read-only, syntax-highlighted view of it. Fuller version:
  browse multiple compose files in the directory and switch the active one.

- [x] **[P] Settings page** — dropped as a page. The tab was a placeholder with
  two rows of content in it, and each of those settings has a better home:
  the compose file is a `--file` flag (per run, explicit), and the theme is a
  picker modal once colors are centralized. What persists goes to
  `~/.config/stack-stitcher/config.yaml` with no page to maintain.

- [ ] **[P] About modal** — the ASCII `LOGO` in `src/constants/Branding.go`
  is explicitly reserved for this. Include version, license, repo link.
  Open with `?` or `a`.

- [ ] **[P] Edit group membership** — follow-up noted in the create/delete
  spec: reopen the `ServiceChecklistModal` for an *existing* group to
  add/remove members (today you can only set membership at creation time or
  delete the whole group).

- [ ] **[P] Group rename** — `DESIGN.md` §3 lists it as unsupported. It's a
  straightforward `yaml.Node` walk (retag every service that carries the
  name); worth doing once membership editing exists.

- [ ] **[P] `--file` / `--directory` flag** — README notes "there's no flag
  to point at a file elsewhere yet". Also unblocks multi-project usage
  without `cd`. Show the active file path in the header or Files page.

## Suggested next steps

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
  `go install github.com/filipemolina/stack-stitcher@latest` can work.
  **Remaining:** version stamping (`-ldflags -X`), `--version`, and showing it
  in the header/About modal — tracked under CI + releases below.

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
  setting. See *Which compose file* in `docs/DESIGN.md`. **Remaining:** when
  several candidate names exist in the directory, mark it (`compose.yaml +2`)
  and list the rest — `GetComposeFileName` returns only the winner today, and
  the natural home for the list is the `?` overlay.

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
  `docs/DESIGN.md`. **Remaining:** the `?` overlay (Phase 5) picks up the
  `alt` aliases and the brackets, which the footer has no room for.

- [ ] **[S] `?` help overlay** — rendered from `src/keys` so it cannot drift
  from the handlers, grouped by scope, unavailable bindings dimmed. Also the
  home for the other compose-file candidates, above, and for the `alt`+letter
  aliases and `[`/`]` brackets, which the footer has no room for. This is
  Phase 5 in `docs/ROADMAP.md`.

- [ ] **[S] The footer wraps on a narrow terminal** — predates the compose
  file segment (which drops itself rather than contributing to this). Below
  roughly 60 columns the context hints plus the global keys exceed the width,
  and the bar wraps to two or three lines, eating body rows. The bar needs to
  shed hints in priority order the way the file name already does. Same
  terminals show two other overflows worth fixing together: the group details
  table collides its column headers (`NAMEIMAGSTATHEALT…`) and the action
  buttons wrap into each other.

- [ ] **[S] Centralize color into a `Theme`** — `src/appstyles/styles.go` has
  good semantic tokens, but they are package-level `var`s built at init, so
  they freeze one palette; five other files still carry stray hexes
  (`#B33A3A`, `#FAFAFA`, `#3F3F3F`). Make them a `Theme` value with a
  registry, which is what a theme picker later needs, and run the existing
  background-bleed suites over every registered theme.

- [ ] **[S] CI + releases** — GitHub Actions: `go build`, `go vet`,
  `go test` on push/PR. Add GoReleaser for tagged releases once the module
  rename lands.

- [ ] **[S] Expand test coverage via the e2e rig** — `src/model/rig_test.go`
  already drives the app in-process (used for the bootstrap flow). Extend
  to: create/delete group flow, docker actions against a fake `docker` on
  `PATH`, and the logs modal.

- [ ] **[S] Logs overlay improvements** — search/filter (`/`), line-wrap
  toggle, toggle timestamps, jump to top (`g`) / bottom (`G`).

- [ ] **[S] Error banner lifecycle** — there is no manual dismissal or
  timeout. A recovered background poll clears its own banner, while other
  errors remain until a successful foreground operation. Add Esc-to-dismiss
  and/or auto-expire after a few seconds.

- [ ] **[S] Re-record `demo/demo.gif`** — the recorded demo predates the
  nav redesign, keybinding bar, and the group vocabulary.

## Housekeeping

- [x] **[H] Document the current build/install path** — README now correctly
  says that `make build` runs `go install .` and installs to
  `$(go env GOPATH)/bin`.

- [ ] **[H] Update `demo/demo.tape`** — its header still refers to
  `dist/stack-stitcher` / `sudo mv`, and its comments still say "profile"
  instead of "group".

- [ ] **[H] Delete `reference/*.go.bak`** — Bubble Tea tutorial leftovers,
  already gitignored; remove from disk.

- [ ] **[H] Fix `terminalWidht` typo** in `configModel`
  (`src/model/AppModel.go`) and its uses in `Update.go` / `View.go`.

- [ ] **[H] Prune stray artifacts** — `vhs-test.gif`/`vhs-test.tape` at the
  repo root look like one-off tests; fold into `demo/` or delete.
