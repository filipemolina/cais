# Edit Existing Services Implementation Plan

> **Status — in progress.** Unchecked boxes below are a live worklist, not a
> historical record. Mark this document completed once every phase is done
> and the TODO item is ticked.

**Goal:** Let the user edit a service by editing its actual YAML — never a
generated form — and splice the result back into the compose file without
disturbing comments, key order, or the spelling of fields they chose.

**Phasing.** Three ways to put the YAML in front of the user, in increasing
difficulty. Each ships something usable and nothing built early is thrown
away later:

| Phase | What it does | Why it survives |
|---|---|---|
| 0 | Preserve the selected service across a config reload | Prerequisite; fixes create/delete group too |
| 1 | `E` opens the whole compose file in `$EDITOR` | Only way to add a service or edit top-level keys |
| 2 | `e` opens just that service in `$EDITOR` | Becomes `ctrl+o` inside the Phase 3 editor |
| 3 | `e` edits the service inline in the details panel | The preferred end state |

Phases 2 and 3 share their hard part: three pure functions in `src/utils/`
(extract a fragment, splice one back, validate the candidate). They are
built once, in Phase 2, and Phase 3 is a second front-end over them.

**Design reference:** `docs/superpowers/specs/2026-07-25-edit-services-design.md`.

## Global constraints

- Module `stack-stitcher`, Go 1.26. Bubble Tea v2 / Bubbles v2 / Lip Gloss
  v2 (`charm.land/...`), `compose-go/v2/types`, `gopkg.in/yaml.v3`.
- Cross-component communication goes through `src/cmds/` message types only.
  Components never call each other directly.
- Compose-file writes go through `writeComposeNode` →
  `utils.ReplaceFileAtomically`. Never `os.WriteFile` (`DESIGN.md`).
- Validation/parse errors from an edit surface through `AppModel.lastError`
  and the existing banner, except where a phase specifies an inline
  presentation.
- One exported constructor per component file, matching the existing
  `Init`/`Update`/`View` shape.
- Each task builds, vets and tests green on its own, and is its own commit.

---

## Phase 0 — Preserve the selection across a reload

Independent of everything else and worth landing first: it fixes
create/delete group today, and without it every later phase looks broken.

### Task 0.1: Re-select the same service and group after a reload

**Files:** `src/model/Update.go`, `src/model/AppModel.go`, new
`src/model/selection_test.go`.

- [ ] **Step 1.** `configSyncCmds` ends with
  `cmds.SetSelectedService(orderedServices[0])` and the equivalent for
  groups. Track the selected service and group names on `AppModel` — they
  are known only to `ServicesList`/`GroupsList` today, so `AppModel` needs
  to observe the `SetSelectedServiceMsg`/`SetSelectedGroupMsg` it already
  routes.
- [ ] **Step 2.** On reload, re-select the tracked name when it still exists
  in the reloaded project; fall back to the first entry when it doesn't
  (removed or renamed outside the app); send nothing when the project is
  empty.
- [ ] **Step 3.** Tests: a reload preserves a mid-list selection; a reload
  after the selection disappears falls back to the first entry; an empty
  project sends no selection message.

**Verify:** `go test ./src/model/`, existing group tests still green.

---

## Phase 1 — The whole compose file in `$EDITOR`

### Task 1.1: Editor resolution

**Files:** new `src/utils/Editor.go`, new `src/utils/Editor_test.go`.

**Interface:** `func EditorCommand(path string) (*exec.Cmd, error)`

- [ ] **Step 1.** Resolve `$VISUAL`, then `$EDITOR`, then `vi`. Split the
  value on whitespace so `EDITOR="code --wait"` works, and build the
  `exec.Cmd` from the parts — **do not** run it through a shell.
- [ ] **Step 2.** Tests with `t.Setenv`: `VISUAL` wins over `EDITOR`; a
  multi-word value becomes command + args; both unset falls back to `vi`;
  a value that is only whitespace is treated as unset.

**Verify:** `go test ./src/utils/`.

### Task 1.2: Suspend the poll while an editor is open

**Files:** `src/model/AppModel.go`, `src/model/Update.go`,
`src/model/refresh_test.go`.

- [ ] **Step 1.** Add an `externalEditorOpen` flag to `AppModel` and a
  condition to `shouldPollContainers()` alongside the existing modal and
  project checks.
- [ ] **Step 2.** Test it the way `refresh_test.go` already tests the modal
  and no-project conditions.

**Verify:** `go test ./src/model/`.

### Task 1.3: Open the compose file, reload on exit

**Files:** new `src/cmds/OpenInEditor.go`, `src/components/DetailsPanel.go`,
`src/model/Update.go`, `src/components/KeybindingBar.go`.

- [ ] **Step 1.** `cmds.OpenInEditor(path string) tea.Cmd` wrapping
  `tea.ExecProcess`, whose callback returns `EditorClosedMsg{Err error}`.
- [ ] **Step 2.** `DetailsPanel` handles `E` in the focused key switch
  alongside `x` and `l`, emitting `cmds.OpenInEditor` for the active compose
  file. The panel doesn't know the file name — carry it in the command from
  `AppModel`, or have the panel emit an intent message `AppModel` turns into
  the command. Prefer the intent message: it keeps the panel ignorant of
  config state, consistent with every other panel key.
- [ ] **Step 3.** `AppModel.Update`: set `externalEditorOpen` when the
  editor opens, clear it on `EditorClosedMsg`, and queue `cmds.GetConfig` so
  the app picks up whatever was saved. An editor that exits non-zero goes to
  the banner.
- [ ] **Step 4.** `KeybindingBar`: `{"E", "edit file"}` on the Dashboard's
  `COMPONENT_BODY_DETAILS` hints only.

**Verify:** `go build ./... && go vet ./... && go test ./...`, then run the
app and edit the file for real — this task is mostly terminal handover,
which no unit test covers convincingly.

---

## Phase 2 — One service in `$EDITOR`

### Task 2.1: Extract a service fragment

**Files:** new `src/utils/ServiceFragment.go`, new
`src/utils/ServiceFragment_test.go`.

**Interface:** `func ExtractServiceFragment(fileName, serviceName string) ([]byte, error)`

- [ ] **Step 1.** `readComposeNode` → `servicesMappingNode` →
  `findMappingValue`. Unknown service errors, matching `AddGroupTag`'s
  wording.
- [ ] **Step 2.** Encode a single-key mapping — the original key node and
  its value node — at indent 2, so the fragment reads exactly as it does in
  the file.
- [ ] **Step 3.** Tests: comments inside the service survive; neighbouring
  services do not appear; an unknown service errors.

**Verify:** `go test ./src/utils/`.

### Task 2.2: Validate a candidate compose document

**Files:** `src/utils/ServiceFragment.go`, tests alongside.

**Interface:** `func ValidateComposeCandidate(dir string, contents []byte) error`

- [ ] **Step 1.** Write `contents` to a temp file in `dir` (not `/tmp` —
  compose resolves build contexts and `env_file:` relative to the compose
  file's directory), run `ReadConfigFile` over it, remove the temp file on
  every path.
- [ ] **Step 2.** Tests: a good document passes; a document whose YAML is
  fine but whose compose is not (e.g. a service that is a string rather than
  a mapping) is rejected with the loader's message; no temp files are left
  behind either way.

**Verify:** `go test ./src/utils/`.

### Task 2.3: Splice an edited fragment back

**Files:** `src/utils/ServiceFragment.go`, tests alongside.

**Interface:** `func ApplyServiceFragment(fileName, serviceName string, fragment []byte) error`

- [ ] **Step 1.** Parse the fragment. Reject: unparseable YAML; a document
  that isn't a single-key mapping; a key that isn't `serviceName` (a rename
  — say so explicitly in the error, and that it isn't supported); a value
  that isn't a mapping.
- [ ] **Step 2.** Replace the value node in the services mapping, keeping
  the original key node so its comments survive.
- [ ] **Step 3.** Encode the document, run `ValidateComposeCandidate` on the
  result, and only then write through `writeComposeNode`. A validation
  failure must leave the file untouched.
- [ ] **Step 4.** Tests: a valid edit rewrites only that service and leaves
  neighbours, key order and their comments intact; each rejection above
  errors *and* leaves the file byte-identical.

**Verify:** `go test ./src/utils/`.

### Task 2.4: Wire `e` to the fragment editor

**Files:** new `src/cmds/EditService.go`, `src/components/DetailsPanel.go`,
`src/model/Update.go`, `src/components/KeybindingBar.go`.

- [ ] **Step 1.** `cmds.EditService` extracts the fragment to a temp file,
  `tea.ExecProcess`es the editor over it, and in the callback reads the file
  back and applies it. Cancel without writing when the bytes are unchanged
  or the file is empty. Remove the temp file on every path.
- [ ] **Step 2.** Result message handled like `CreateGroupMsg`: error to the
  banner, success queues `cmds.GetConfig`. A parse or validation failure is
  an ordinary error — the file is untouched, the user is back in a normal
  TUI, and pressing `e` again is the retry. There is no editor loop and no
  mode to escape. Make the banner carry the loader's own message, since
  "invalid compose file" alone doesn't help anyone fix it.
- [ ] **Step 3.** `DetailsPanel` handles `e`; `KeybindingBar` gains
  `{"e", "edit"}` on the Dashboard details hints.
- [ ] **Step 4.** An end-to-end test with `$EDITOR` set to a shell script
  that rewrites the fragment, asserting the compose file afterwards.

**Verify:** `go build ./... && go vet ./... && go test ./...`.

### Task 2.5: Drafts — resume a rejected edit

**Deferred.** Phase 2 is complete and usable without this: a failed
validation already reports the error, leaves the file untouched, and drops
the user back into a normal TUI (Task 2.4, Step 2). This task only removes
the sting of retyping a substantial edit, and can land any time afterwards.

Explicitly **not** a `visudo`-style reopen loop. Leaving must never require
fixing the text first.

**Files:** new `src/utils/Draft.go`, tests alongside, `src/cmds/EditService.go`.

- [ ] **Step 1.** Store a rejected fragment under
  `$XDG_CACHE_HOME/stack-stitcher/drafts/` (falling back to
  `~/.cache/...`), keyed by the compose file's absolute path and the service
  name. Never beside the compose file — that puts junk in the user's repo.
- [ ] **Step 2.** Store the fragment the draft was based on alongside it.
- [ ] **Step 3.** On `e`, if a draft exists and its base still matches the
  service's current YAML, open the draft and tell the user they're resuming
  one. If the base no longer matches, the file changed underneath: say so,
  start from the current file, and leave the draft in place rather than
  destroying it. Silently resuming a stale draft would revert whatever
  changed in between.
- [ ] **Step 4.** Remove the draft on a successful save, and offer an
  explicit discard.
- [ ] **Step 5.** Tests: a rejected edit leaves a draft; the next edit
  resumes it; a successful save clears it; a draft whose base has drifted is
  not resumed silently.

**Verify:** `go test ./...`.

---

## Phase 3 — Inline in the details panel

### Task 3.1: Edit mode in the details panel

**Files:** `src/components/DetailsPanel.go`, new
`src/components/ServiceEditor.go`.

- [ ] **Step 1.** `e` swaps the rendered card for a `textarea`
  (`charm.land/bubbles/v2/textarea`) holding the Phase 2 fragment.
- [ ] **Step 2.** **Gate the panel's key handling on edit mode before
  anything else.** `s`, `t`, `r`, `p`, `x` and `l` are docker actions on
  this panel; while editing they must be plain text. Typing `ports:` must
  not pull and stop the service, and `x` must not open a
  container-destroying confirmation.
- [ ] **Step 3.** `ctrl+s` saves through `ApplyServiceFragment`. `esc`
  cancels, confirming first when the text has changed — the only place in
  the app where `esc` could discard real work. `ctrl+o` hands off to the
  Phase 2 `$EDITOR` path for a fragment too big for the panel.
- [ ] **Step 4.** A test that edit mode swallows every action key.

**Verify:** `go test ./src/model/ ./src/components/`.

### Task 3.2: Live validation

**Files:** `src/components/ServiceEditor.go`.

- [ ] **Step 1.** Parse the buffer on change and show YAML syntax status on
  a line under the editor. The fragment is small; no debounce needed until
  measurement says otherwise.
- [ ] **Step 2.** On save, run the full compose validation and keep the
  editor open with the error when it fails.

**Verify:** `go test ./...`.

### Task 3.3: Decide the editor's width

- [ ] **Step 1.** Use it. The details panel is the right-hand half of the
  body, which may be too narrow for YAML.
- [ ] **Step 2.** If it is, let the editor take the full body width while
  active and restore the split on exit. **Do not build this speculatively** —
  the design deliberately leaves it open.

---

## Task 4: Documentation

**Files:** `README.md`, `docs/DESIGN.md`, `TODO.md`, this plan, the design.

- [ ] **Step 1.** README: `e` and `E` in the keybindings table; replace
  "Editing existing services is still on the roadmap" with what actually
  ships.
- [ ] **Step 2.** `DESIGN.md`: record that service editing is YAML-based
  rather than a form and why, that edits go through the same atomic path,
  and that a file edit never touches a running container.
- [ ] **Step 3.** `TODO.md`: tick the item, and record what was
  deliberately left out — service rename, reacting to on-disk changes,
  structured add/delete of services.
- [ ] **Step 4.** Mark this plan and its design completed, matching the
  convention of the other documents in these directories.

**Verify:** `go build ./... && go vet ./... && go test ./...` a final time.
