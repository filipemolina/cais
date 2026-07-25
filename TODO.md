# TODO

Working list for Stack Stitcher. Sources: the roadmap in `README.md`, the
guiding principles in `docs/DESIGN.md`, the "Out of scope / follow-ups"
sections of the specs in `docs/superpowers/specs/`, plus review findings.

Legend: **[P]** = from the original plan/roadmap, **[S]** = suggested next
step, **[H]** = housekeeping.

`README.md`, `docs/DESIGN.md`, and this file are the current documentation.
The dated specs and plans under `docs/superpowers/` are completed historical
records, not a live backlog.

---

## Remaining from the original plan

- [ ] **[P] Edit existing services** — the one remaining roadmap item
  (README: "Editing existing services is still on the roadmap"). Per
  `DESIGN.md` §6.3 it lands on the **Dashboard** first, not Home. Natural
  first fields: image tag, ports, env vars. Should reuse the
  comment-preserving `yaml.Node` edit path in `src/utils/GroupTags.go`.
  Design and plan written (2026-07-25), phased to land the image field
  first: [design](docs/superpowers/specs/2026-07-25-edit-services-design.md),
  [plan](docs/superpowers/plans/2026-07-25-edit-services.md). These two are
  **live**, unlike the completed records beside them.

- [x] **[P] Atomic compose-file writes** — `utils.ReplaceFileAtomically`
  writes a temporary file alongside the target and renames it into place,
  preserving the original's permissions. A failed write now leaves the
  compose file untouched instead of truncated. Done before the
  edit-services feature, which makes writes routine.

- [ ] **[P] Compose Files page** — currently a `PlaceholderPanel`. The tab
  label is already "Files". Minimum useful version: show which compose file
  is loaded and a read-only, syntax-highlighted view of it. Fuller version:
  browse multiple compose files in the directory and switch the active one.

- [ ] **[P] Settings page** — currently a `PlaceholderPanel` ("Colors, key
  bindings and the default compose file will live here"). Persist to a
  config file (e.g. `~/.config/stack-stitcher/config.yaml`).

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

- [ ] **[S] Rename the module for distribution** — `go.mod` says
  `module stack-stitcher`, so `go install
  github.com/filipemolina/stack-stitcher@latest` can't work. Rename the
  module to the repo path, then add version stamping (`-ldflags -X`) and
  show it in the header/About modal.

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
