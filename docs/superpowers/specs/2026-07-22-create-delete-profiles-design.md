# Create/Delete Profiles — Design

## Context

Stack Stitcher's README roadmap lists "creating/deleting profiles from the TUI" as an open item. Compose "profiles" group related services so they can be started/stopped together; today the app can only read profiles (derived from each service's `profiles:` field) and act on them — it can't create or remove one.

A profile is not a first-class object in `compose.yml`. It only exists as a `profiles:` tag on individual services — `configSyncCmds` in `src/model/Update.go` derives the visible profile list purely by scanning every service's `Profiles` field. This shapes the whole feature: "creating a profile" means tagging one or more existing services with a new name, and "deleting a profile" means stripping that tag from every service that has it. No services are created or deleted by this feature.

## Goals

- Let the user create a new profile by naming it and selecting which existing services belong to it.
- Let the user delete a profile, removing the tag from every service that has it, after confirming.
- Persist both changes to the on-disk `compose.yml`, preserving the file's existing formatting/comments as much as possible.
- Keep the diff small and in the existing architectural style (per-component `Init`/`Update`/`View`, `cmds/` messages, `componentId`-based focus) — this is the user's first Go project and they want to keep recognizing the codebase.

## Non-goals

- Editing other service fields (image, ports, env, etc.) — that's the separate "edit services" roadmap item.
- Bootstrapping a `compose.yml` from scratch — separate roadmap item.
- A generic/reusable modal framework beyond what this feature needs.

## Architecture: introducing a modal overlay

This is the first modal/overlay in the app. Today `AppModel.View()` (`src/model/View.go`) always composes the same fixed panels for the active page, and `AppModel.ChangeFocus` (`src/model/AppModel.go`) cycles Tab focus only through `constants.FocusableComponents`. There's no concept of a transient UI element on top of that.

Proposed addition: an optional `activeModal tea.Model` field on `AppModel`.

- When `activeModal != nil`, `AppModel.Update` routes `tea.KeyPressMsg` (and other relevant messages) to the modal only — Tab/Shift+Tab focus-cycling and the underlying panels' own key handling are bypassed until the modal closes. Non-key messages (e.g. `tea.WindowSizeMsg`) still propagate normally so the modal can size itself.
- When `activeModal != nil`, `AppModel.View` renders the modal centered over the normal layout using `lipgloss.Place`, instead of (or on top of) the regular page body.
- A modal closes itself by returning a sentinel `cmds.CloseModalMsg` (optionally carrying a follow-up `tea.Cmd` to run, e.g. `cmds.CreateProfile(...)`), which `AppModel.Update` handles by clearing `activeModal` and appending the follow-up cmd to `finalCmds`.

This is a real structural addition (new field, new branch in `Update`/`View`, bypass of the focus system), not just a new leaf component — future modal-shaped features (e.g. editing a service) can reuse the same `activeModal` mechanism instead of inventing a new one.

## Components

All new files live under `src/components/`, following the existing one-file-per-component convention.

### `ProfileNameModal`
- Step 1 of the create flow. A `bubbles/textinput`-based prompt: "New profile name:".
- Validation on submit: non-empty, and not already present in the current profiles list (case-sensitive match). Invalid input shows an inline message inside the modal — this is a form validation error, not a docker/IO failure, so it does **not** go through the app-wide `m.lastError` banner.
- `Enter` with valid input → emits a transition to `ServiceChecklistModal` (carrying the chosen name forward).
- `Esc` → emits `cmds.CloseModalMsg` with no follow-up (cancels the whole create flow).

### `ServiceChecklistModal`
- Step 2 of the create flow. A `bubbles/list.Model` with a custom delegate rendering each service with a `[ ]`/`[x]` prefix (visually consistent with `ProfilesListCustomDelegate`'s existing active/selected states, but adds the checkbox glyph).
- `Space` toggles the highlighted service's checked state (does not move selection, mirrors how `ServicesList`/`ProfilesList` already use Space for "select" vs. arrow keys for "move").
- `Enter` requires at least one checked service; emits `cmds.CloseModalMsg` with a follow-up `cmds.CreateProfile(name, checkedServiceNames)`.
- `Esc` → emits `cmds.CloseModalMsg` with no follow-up (cancels the whole create flow, including the name already entered).

### `ConfirmModal`
- Generic yes/no prompt: renders a message (e.g. `Delete profile "web"? (y/n)`) and reacts to `y`/`n`/`Esc`.
- Reusable: constructed with a message string and a `tea.Cmd` to run on confirm. Used here for delete; not over-engineered beyond what delete needs (no title bar, no multi-button layout).
- `y` → emits `cmds.CloseModalMsg` with the confirm cmd as follow-up.
- `n` / `Esc` → emits `cmds.CloseModalMsg` with no follow-up.

## Keybindings

Added to the Home page's Groups panel (`src/components/ProfilesList.go`), matching the existing `s/t/r/p/x` action-key pattern already scoped to a focused panel:

| Key | Action | Where |
| --- | --- | --- |
| `n` | Open create-profile modal | Groups panel focused |
| `d` | Open delete-profile confirmation for the highlighted profile | Groups panel focused |

## Data flow

**Create:**
1. `n` on Groups panel → `AppModel` sets `activeModal = ProfileNameModal(existingProfileNames)`.
2. Name confirmed → `activeModal = ServiceChecklistModal(name, allServiceNames)`.
3. Services confirmed → `cmds.CloseModalMsg{Follow: cmds.CreateProfile(name, serviceNames)}`.
4. `cmds.CreateProfile` (new, in `src/cmds/CreateProfile.go`) calls new `utils.AddProfileTag(composePath, name, serviceNames) error`.

**Delete:**
1. `d` on Groups panel (with a profile selected) → `AppModel` sets `activeModal = ConfirmModal("Delete profile \"<name>\"? (y/n)", cmds.DeleteProfile(name))`.
2. `y` → `cmds.CloseModalMsg{Follow: cmds.DeleteProfile(name)}`.
3. `cmds.DeleteProfile` (new, in `src/cmds/DeleteProfile.go`) calls new `utils.RemoveProfileTag(composePath, name) error`.

**Shared write path (`src/utils/ProfileTags.go`, new):**
- `AddProfileTag`/`RemoveProfileTag` read the raw compose file bytes and parse them into a `yaml.v3` `*yaml.Node` document (`go.yaml.in/yaml/v4`'s sibling `gopkg.in/yaml.v3` is already an indirect dependency via `compose-go`, so this adds no new module — using `gopkg.in/yaml.v3` directly since it's the one with the documented Node API).
- Walk: root → mapping → `services` key's mapping → `<serviceName>` key's mapping → `profiles` key's sequence node (create the sequence, and the `profiles:` key, if it doesn't exist yet, for `AddProfileTag`).
- `AddProfileTag`: for each target service, append a scalar node with `name` unless already present (idempotent — no duplicate tags).
- `RemoveProfileTag`: for every service, if it has a `profiles` sequence containing `name`, remove that scalar; if the sequence becomes empty, remove the `profiles` key entirely from that service's mapping (don't leave `profiles: []` behind).
- Re-encode the modified node tree with `yaml.v3`'s `Encoder` and write it back to the same path (same file, not a temp-then-rename — matches the simplicity of the rest of the app's file handling; no other write path in the codebase does atomic replace).
- This preserves comments, key ordering, and untouched sections of the file — unlike `Project.MarshalYAML()`, which would regenerate the entire document from the parsed struct and drop all of that.

**Refresh:**
- Both `cmds.CreateProfile` and `cmds.DeleteProfile` return a msg (`CreateProfileMsg{Err error}` / `DeleteProfileMsg{Err error}`) handled in `AppModel.Update` the same way `cmds.DockerActionMsg` is handled today: on success, append `cmds.GetConfig` to `finalCmds` so the project is re-parsed from disk and `configSyncCmds()` re-derives `ProfilesList` / `GroupDetailsPanel` from the updated file. There's no separate in-memory profile list to keep manually in sync — disk is the source of truth, same as it is today.

## Error handling

- **Form validation** (empty/duplicate name, zero services checked): handled inline inside the modal, blocks submission, never reaches `AppModel` or the docker/IO layer.
- **File IO / YAML walk failures** (e.g. compose file disappeared, unexpected structure): `AddProfileTag`/`RemoveProfileTag` return a wrapped `error`; `CreateProfileMsg.Err`/`DeleteProfileMsg.Err` carries it into `m.lastError`, rendered by the existing red banner in `src/model/View.go` — identical to how `DockerActionMsg` errors surface today.

## Testing

No `*_test.go` files exist anywhere in this repo today, and this design doesn't add a test suite for the whole feature — the modals, cmds, and wiring follow the existing (untested) convention. The one exception is `src/utils/ProfileTags.go`: it's pure logic with no TUI/Docker dependency, and a bug in it corrupts the user's `compose.yml`, so it gets a focused `ProfileTags_test.go` covering:
- Adding a tag to a service with no existing `profiles:` key.
- Adding a tag to a service that already has a `profiles:` list (appends, doesn't duplicate).
- Adding a tag already present (no-op, idempotent).
- Removing the only tag on a service (drops the `profiles:` key entirely).
- Removing one of several tags (leaves the others, and the key, intact).

## Out of scope / follow-ups

- Editing which services belong to an *existing* profile after creation (add/remove members later) isn't covered — today's design only covers create-time selection and full delete. Could be a natural follow-up once "edit services" lands, since both need a service-membership editor.
- Atomic file write (temp file + rename) isn't included since nothing else in the codebase does it either; worth reconsidering if this pattern gets reused for the "edit services" and "bootstrap compose file" roadmap items, where write failures would be more consequential.
