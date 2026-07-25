# Edit Existing Services Implementation Plan

> **Status — in progress.** Unchecked boxes below are a live worklist, not a
> historical record. Mark this document completed once every task is done and
> the TODO item is ticked.

**Goal:** Let the user change a service's image from the Dashboard's details
panel, written through the existing comment-preserving, atomic compose-file
path, with the panel showing the result afterwards.

**Architecture:** The path established by create/delete-groups, unchanged:
keybinding on a panel → `cmds.Open*Modal` → `AppModel.activeModal` → modal
validates → `cmds.CloseModal(follow)` → command calls a pure function in
`src/utils/` → result message reloads config from disk. The only structural
change is in `configSyncCmds`, which must stop resetting the selection on
every reload.

**Scope:** Image only. Ports and env vars are deliberately out of this
phase — see "Why image first" in the design.

**Design reference:** `docs/superpowers/specs/2026-07-25-edit-services-design.md`.

## Global constraints

- Module `stack-stitcher`, Go 1.26. Bubble Tea v2 / Bubbles v2 / Lip Gloss v2
  (`charm.land/...`), `compose-go/v2/types`, `gopkg.in/yaml.v3`.
- Cross-component communication goes through `src/cmds/` message types only.
  Components never call each other directly.
- Form-validation errors stay inline in the modal; IO errors surface through
  `AppModel.lastError` and the existing banner.
- Compose-file writes go through `writeComposeNode` →
  `utils.ReplaceFileAtomically`. Never `os.WriteFile` (`DESIGN.md`).
- One exported constructor per component file, matching the existing
  `Init`/`Update`/`View` shape.
- Each task should build, vet, and test green on its own, and be its own
  commit.

---

### Task 1: Preserve the selected service across a config reload

Independent of the rest and worth landing first — it fixes create/delete
group today, and without it the edit feature looks broken.

**Files:** `src/model/Update.go`, `src/model/AppModel.go` (if the selection
needs to be tracked), new `src/model/selection_test.go`.

- [ ] **Step 1.** `configSyncCmds` currently ends with
  `cmds.SetSelectedService(orderedServices[0])` and the same for groups.
  Track the currently selected service name on `AppModel` (it is only known
  to `ServicesList` today) and re-select it when it still exists in the
  reloaded project, falling back to `orderedServices[0]` when it doesn't.
- [ ] **Step 2.** Do the same for the selected group, so create/delete group
  stops jumping to the top of the list.
- [ ] **Step 3.** Tests: a reload preserves a mid-list selection; a reload
  after the selected service disappears falls back to the first entry; an
  empty project sends no selection message at all.

**Verify:** `go test ./src/model/`, plus existing group tests still green.

---

### Task 2: `utils.SetServiceImage`

**Files:** `src/utils/GroupTags.go` (or a new `src/utils/ServiceFields.go` if
it reads better — the file is named for group tags and this isn't one),
`src/utils/ServiceFields_test.go`.

**Interface:** `func SetServiceImage(fileName, serviceName, image string) error`

- [ ] **Step 1.** Read the doc with `readComposeNode`, get the services
  mapping with `servicesMappingNode`, find the service with
  `findMappingValue`. Unknown service → error naming it, matching
  `AddGroupTag`'s wording.
- [ ] **Step 2.** If an `image:` key exists, assign the new value to the
  existing scalar node — do not replace the node, so its quoting style
  survives. If it doesn't exist, append a key/value pair the way
  `AddGroupTag` appends `profiles:`.
- [ ] **Step 3.** Write via `writeComposeNode`.
- [ ] **Step 4.** Tests, using a fixture that carries inline comments:
  replaces an existing image; leaves comments, key order and unrelated
  services untouched; appends the key when absent; errors on an unknown
  service.

**Verify:** `go test ./src/utils/`.

---

### Task 3: The command layer

**Files:** new `src/cmds/SetServiceImage.go`, new
`src/cmds/OpenEditServiceModal.go`.

- [ ] **Step 1.** `SetServiceImage(serviceName, image string) tea.Cmd`
  returning `SetServiceImageMsg{Err error}` — mirrors `CreateGroup.go`,
  including the `utils.GetComposeFileName()` lookup.
- [ ] **Step 2.** `OpenEditServiceModal(service types.ServiceConfig) tea.Cmd`
  returning `OpenEditServiceModalMsg` — mirrors `OpenConfirmModal.go`. It
  carries the service so the modal can prefill without reaching into
  `AppModel`.

**Verify:** `go build ./...`.

---

### Task 4: `EditServiceModal`

**Files:** new `src/components/EditServiceModal.go`.

- [ ] **Step 1.** A single `textinput` prefilled with the current image,
  focused, cursor at end (`CursorEnd()`, as `CreateComposeFileModal` does
  for the filename).
- [ ] **Step 2.** Keys: `enter` validates and emits
  `cmds.CloseModal(cmds.SetServiceImage(name, image))`; `esc` emits
  `cmds.CloseModal(nil)`. Everything else forwards to the input, including
  non-key messages so the cursor keeps blinking.
- [ ] **Step 3.** Validation: non-empty, no whitespace. Nothing stricter —
  see the design. Errors render inline via the existing `errStyle`
  convention.
- [ ] **Step 4.** View through `modalSurface`, with the service name in the
  title and the footer line
  `Applies on next start (s) — restart won't recreate the container.`

**Verify:** `go build ./...`.

---

### Task 5: Wire it up

**Files:** `src/components/DetailsPanel.go`, `src/model/Update.go`,
`src/components/KeybindingBar.go`.

- [ ] **Step 1.** `DetailsPanel`: handle `e` in the existing focused-and-
  selected key switch, alongside `x` and `l`, emitting
  `cmds.OpenEditServiceModal(*m.service)`.
- [ ] **Step 2.** `AppModel.Update`: `OpenEditServiceModalMsg` sets
  `activeModal`, exactly like `OpenConfirmModalMsg`.
- [ ] **Step 3.** `AppModel.Update`: `SetServiceImageMsg` handled like
  `CreateGroupMsg` — clear `lastErrorFromPoll`, error to the banner or
  success queues `cmds.GetConfig`.
- [ ] **Step 4.** `KeybindingBar`: add `{"e", "edit"}` to the Dashboard's
  `COMPONENT_BODY_DETAILS` hints only, not Home's.

**Verify:** `go build ./... && go vet ./... && go test ./...`.

---

### Task 6: End-to-end test

**Files:** `src/model/edit_service_test.go`.

- [ ] **Step 1.** Using the in-process rig (`rig_test.go`), in a temp
  directory with a fixture compose file: select a service, focus the details
  panel, press `e`, type a new image, press enter.
- [ ] **Step 2.** Assert the file on disk has the new image and its comments
  intact, and that the details panel is still showing the edited service
  (which is Task 1's behavior, verified end to end here).
- [ ] **Step 3.** Assert `esc` writes nothing.

**Verify:** `go test ./...`.

---

### Task 7: Documentation

**Files:** `README.md`, `docs/DESIGN.md`, `TODO.md`, this plan, the design.

- [ ] **Step 1.** README: add `e` to the keybindings table; soften the
  "Editing existing services is still on the roadmap" line to say that the
  image can now be edited and that ports/env are next.
- [ ] **Step 2.** `DESIGN.md`: note in the state/destructive-actions section
  that service edits go through the same atomic path, and that a file edit
  does not touch a running container.
- [ ] **Step 3.** `TODO.md`: tick the edit-services item for image, and add
  ports and env vars as their own follow-up entries with the syntax-
  preservation problem recorded, so the reasoning isn't lost.
- [ ] **Step 4.** Mark this plan and its design completed, matching the
  convention of the other documents in these directories.

**Verify:** `go build ./... && go vet ./... && go test ./...` a final time.
