# Bootstrap a Compose File — Design

> **Status — implemented / historical.** The bootstrap modal now opens
> automatically when no compose file exists, and the behavior is covered by
> `src/model/bootstrap_test.go`. Do not treat the historical "today"
> statements below as current behavior. See [README](../../../README.md),
> [Design](../../DESIGN.md), and [TODO](../../../TODO.md) for the current
> state and backlog.

## Context

Stack Stitcher's README roadmap lists "bootstrapping a `compose.yml` from scratch" as an open item. Today, running `stack-stitcher` in a directory without a compose file (`utils.GetComposeFileName` — `src/utils/GetComposeFileName.go`) returns `"no compose.yaml, compose.yml, docker-compose.yaml or docker-compose.yml found in the current directory"`, which surfaces in `m.lastError` and leaves the user staring at an empty, non-interactive error banner. The only way out is to quit, hand-write YAML, and re-run.

The user has create/delete profiles working (`n`/`d` on the Groups panel), but those features mutate an *existing* `compose.yml` — they don't help when there is no file at all. "Bootstrap" closes that last gap so the whole flow (read → group → operate → log) can be exercised from a clean directory.

## Goals

- Let the user create a new `compose.yaml` in the current working directory from inside the TUI, without leaving it.
- Get the new file into a state Stack Stitcher can already load: at minimum, a top-level `services:` mapping (possibly with one user-supplied first service so something actually shows up in the Dashboard).
- After creation, the existing `GetConfig` → `configSyncCmds` refresh path picks up the new file with no other code changes — disk stays the source of truth, same as for the create/delete-profiles work.

## Non-goals

- A full visual compose-file editor (image / ports / env / volumes / networks on every service). That's the "edit services" roadmap item — this feature only writes the file with whatever fields the user explicitly types in, nothing more.
- Auto-detecting / merging with an existing file the user didn't notice. If a file already exists, the modal refuses to overwrite it and tells the user so.
- Multi-service bootstrap. One optional first service is enough to get a useful Dashboard view; the user can then add more via the create-profile flow + (future) edit-services work.
- Touching anything outside the current working directory. No path picker, no `--file` flag — the existing `GetComposeFileName` precedence is what the user already runs against.

## UX: where the bootstrap lives

Two states the user can be in on startup:

1. **No compose file present.** This is the new case. On startup, when `cmds.GetConfig` returns the new `utils.ErrNoComposeFile` sentinel, `AppModel.Update` automatically opens the `CreateComposeFileModal` — the user is dropped straight into the form, before any other interaction. If they Esc the modal, the existing red error banner shows the underlying message so they understand the state (no compose file, no profiles, no services to operate on).
2. **Compose file present (or any other error).** Behavior is unchanged. Existing `GetConfig` errors still surface through `m.lastError`; only the missing-file case changes.

The trigger is the `ErrNoComposeFile` sentinel itself, scoped to the `GetConfigMsg` handler — not a keypress, not a per-page state. This is the only place in the app where a modal opens in response to an error rather than an explicit user action; the rationale is that there is no useful state to show the user until a compose file exists, so the only productive next step is to offer to make one. The user can still dismiss (Esc) and see the normal error banner.

## Components

All new files live under `src/components/`, following the existing one-file-per-component convention and reusing the `activeModal` infrastructure added by the create/delete-profiles work (`src/model/Update.go`, `src/model/View.go`).

### `CreateComposeFileModal`

A two-step form, single modal that internally steps between two panes, mirroring the `ProfileNameModal` → `ServiceChecklistModal` pattern from create/delete:

**Step 1 — Filename.** A `bubbles/textinput` prompt for the file to create, prefilled with `compose.yaml`. Validation:
- Non-empty.
- Must end in `.yaml` or `.yml`.
- Must not already exist in the current directory (otherwise we tell the user to pick another name and point them at the existing file).
- Enter → advance to step 2. Esc → close modal with no follow-up.

**Step 2 — Optional first service.** Asks "Add a first service?" with two soft buttons (`y`/`n` / arrows / Enter). If `y`, two more text inputs: service name (required, must be a valid identifier and not already present in the new file) and image (required, e.g. `nginx:alpine`). If `n`, the file is created with just an empty `services:` mapping.

- Enter on the final input / button → emits `cmds.CloseModalMsg{Follow: cmds.CreateComposeFile(filename, optionalServiceName, optionalImage)}`.
- Esc at any point in step 2 → close modal with no follow-up (the file is **not** created; we never half-create).

Validation errors (invalid filename, name already in use, empty service/image fields) render inline inside the modal — same convention as `ProfileNameModal`. They never reach `AppModel` or the docker/IO layer.

### Reuse

- The existing `activeModal` overlay, key-routing bypass, and `cmds.CloseModalMsg` follow-up mechanism from create/delete-profiles.
- The existing `appstyles` and Lip Gloss patterns (rounded border, primary-color border, `PanelBackgroundColor`) from `ConfirmModal`/`ProfileNameModal`/`ServiceChecklistModal`.
- The `SetActivePage` / `GetConfig` refresh cycle — after the file is written, `cmds.CreateComposeFile` returns success and `AppModel.Update` runs `cmds.GetConfig` exactly like `CreateProfileMsg`/`DeleteProfileMsg` do today.

## Keybindings

None. The bootstrap modal opens automatically on startup when no compose file is present, so the user never has to remember a key. Esc dismisses the modal (revealing the error banner); the next session's startup will offer the modal again.

## Data flow

**Trigger:**
1. On startup, `cmds.GetConfig` returns `Err: ErrNoComposeFile` (a new sentinel exported from `src/utils/GetComposeFileName.go`).
2. `AppModel.Update`'s `GetConfigMsg` handler inspects the error with `errors.Is`. If it's `ErrNoComposeFile`, it sets `activeModal = components.CreateComposeFileModal()` and *also* populates `m.lastError` (so an Esc from the modal still leaves a visible explanation).
3. After the user completes or cancels the modal, the existing refresh path takes over (see below). No keypress required.

**Form → write:**
1. On `GetConfigMsg` with `ErrNoComposeFile`, `AppModel.activeModal = components.CreateComposeFileModal()`.
2. User fills in filename + optional service/image → `activeModal` emits `cmds.CloseModalMsg{Follow: cmds.CreateComposeFile(filename, name, image)}`.
3. `cmds.CreateComposeFile` (new, in `src/cmds/CreateComposeFile.go`) calls new `utils.WriteNewComposeFile(fileName, optionalService)` and returns `CreateComposeFileMsg{Err error}`.

**Write path (`src/utils/ProfileTags.go` already exists — extend it, don't fork):**
- Add `WriteNewComposeFile(fileName string, serviceName, image string) error` to the same file as `AddProfileTag`/`RemoveProfileTag` (it's the same family — "edit a `compose.yml` on disk in a comment-preserving way").
- Build a `yaml.Node` document with top-level `services:` mapping, plus an optional `services.<name>:` mapping with `image: <image>` if the user supplied a service.
- Re-encode with the same `yaml.NewEncoder` / `SetIndent(2)` setup that `writeComposeNode` already uses.
- Reject if the target path already exists (return a wrapped `os.ErrExist` so the modal can show a clear message; the existence check happens before any parse step, since the file is being created from scratch).
- The file gets `0o644`, matching `writeComposeNode`.

**Refresh:**
- On success, `AppModel.Update` appends `cmds.GetConfig` to `finalCmds`, same as `CreateProfileMsg`/`DeleteProfileMsg`/`DockerActionMsg` do today. The user lands on the Home page, now with whatever the new compose file contains.

## Error handling

- **Form validation** (bad filename, duplicate service name, empty image): inline inside the modal, never reaches `AppModel`.
- **`ErrNoComposeFile` on startup**: `AppModel.Update` auto-opens the bootstrap modal **and** populates `m.lastError`. If the user Escs the modal, the error banner remains visible so the state is explained.
- **File IO failure** (permission denied, disk full, race where the file appeared between our check and the write): `WriteNewComposeFile` returns a wrapped error, `CreateComposeFileMsg.Err` carries it, `AppModel.lastError` displays it through the existing red banner. Same path as every other IO error in the app.

## Testing

Same convention as the existing create/delete work: only the pure-logic file-getting-changed (`ProfileTags.go`) gets a `*_test.go`, and the new tests live next to the existing `ProfileTags_test.go` cases (a new sibling test file is fine if the focus is on `WriteNewComposeFile`; we add cases to `ProfileTags_test.go` if it stays cohesive). Cases:

- `WriteNewComposeFile` with no optional service creates a file with an empty `services:` mapping that round-trips through `compose-go`.
- `WriteNewComposeFile` with `serviceName="app"` and `image="nginx:alpine"` produces a file that loads as a one-service project and is then immediately editable by `AddProfileTag`.
- `WriteNewComposeFile` against an existing file returns an error and does not modify the existing file.
- (Stretch) The newly created file then loads through the existing `utils.ReadConfigFile` and the resulting `*types.Project` has the expected service.

Components, cmds, and wiring follow the existing untested convention — checked with `go build ./...` plus the manual verification pass.

## Out of scope / follow-ups

- "Pick a different directory" / `--file` flag — separate concern, the `GetComposeFileName` precedence already locks us to CWD.
- Editing services after creation (image / ports / env / volumes) — that's the separate "edit services" roadmap item; bootstrap only writes the file with the minimum the user explicitly asked for.
- An "open existing file" picker once the user has multiple compose files in different directories — out of scope here, would be a different feature.
