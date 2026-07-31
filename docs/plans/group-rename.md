# Group Rename — Implementation Plan

## Status of the feature

**Not implemented.** Verified by:

- `TODO.md` line 100: `- [ ] **[P] Group rename**` — open item.
- `docs/DESIGN.md` §3: "**Renaming a group** is currently not supported (would
  require multi-file YAML rewriting)."
- `docs/ROADMAP.md` §"Explicitly post-alpha": "Group rename" listed as future.
- No `RenameGroup*` symbol anywhere in `src/` (grep). The closest existing
  thing is `EditGroup` / `ServiceChecklistModalForEdit`, which edits a group's
  **membership** (which services carry the tag), not its name.

## Problem

A group is not a first-class object in `compose.yml` — it exists only as a
`profiles:` string tag on individual services (`docs/DESIGN.md` §3). The
groups list is derived from `allGroupNames()`, which collects every distinct
profile across the loaded project. Today the user can create a group
(`n`), edit its membership (`e`), and delete it (`d`), but the only way to
rename it is to edit the YAML by hand (`E`), which is a multi-spot edit with
no validation. The UI verb for that action does not exist.

Renaming is the one group operation DESIGN.md calls out as unsupported, and
TODO.md explicitly says it is "a straightforward `yaml.Node` walk (retag
every service that carries the name); worth doing once membership editing
exists." Membership editing (Phase 8) has landed. This is that item.

## Solution

`R` on the groups list opens a rename modal pre-filled with the group's
current name. Enter writes the rename; Esc cancels. The write is a single
read-modify-write pass over the loaded compose file that replaces every
`profiles:` entry equal to the old name with the new name — a pure value
replacement, exactly the `yaml.Node` walk TODO.md sketches.

### UX: the key

`R` (shift+r) on the groups list, declared once in `keys.List` as `Rename`,
handled only by `GroupsList` (the services list is read-only, so `R` does
nothing there — same as `n`/`e`/`d` today).

Why `R`:

- **Unbound everywhere** — verified by grep: no binding uses `R` in
  `Global`, `List`, `Details`, `Editor`, `Files`, or `Overlay`.
- **Uppercase-on-list has precedent** — the bubbles list already binds `G`
  (go to end) via `ListKeyMap`. Uppercase keys are context-scoped in this
  app (`T` theme, `E` file), so `R` on the list while `r` means *restart* on
  the details panel is the same "different panel, different verb" shape as
  `e`/`E` (edit service vs edit whole file). The one-verb-one-binding rule
  is about the *same* verb; restart and rename are different verbs.
- Help text "R rename" reads unambiguously, unlike reusing `E` (which would
  collide conceptually with the details panel's `E file` and give the list
  an `e`/`E` pair with different meanings per letter).

**Decision owner:** the key choice is a maintainer call. `E` on the list
(edit name, pairing with `e` = edit membership) is the runner-up; `R` is
recommended because `E` already means "file" in two other contexts and the
help text would lie ("E rename" while `E` is "file" everywhere else).

The key is a **primary group-management verb** (like `n`/`e`/`d`), so it
goes in the footer via `keys.Active` — unlike the global keys `a`/`T`,
which are deliberately help-overlay-only. Trade-off: the Home list footer
grows by one hint, and the footer's narrow-terminal wrapping is already an
open TODO item; this slightly worsens that case. If the maintainer prefers,
the fallback is to leave `R` out of `Active()` and let `?` be the only
advertisement (one line to revert).

### UX: the modal

Reuse `GroupNameModal` with a rename mode, mirroring how
`ServiceChecklistModal`/`ServiceChecklistModalForEdit` share one component
behind an `isEdit` flag. The create flow's step 1 is already "a text input
that validates a unique, non-empty group name" — the rename flow is the same
input with a different submit action and title. Keeping the validation in
one place is the point: two modals validating group names would drift.

Differences in rename mode:

- Title `Rename group` (vs `New group`); Enter hint `rename` (vs `next`).
- Input **pre-filled** with the current name, cursor at end. Editing a name
  is usually a small change (`core` → `core-prod`); forcing a retype would
  be hostile. Note: bubbles v2 `textinput` has no select-all — `ctrl+a` is
  cursor-to-line-start and `ctrl+u` clears the field. Users who want to
  replace wholesale can `ctrl+u` then type. This is a known, accepted
  limitation of the input widget, not something this plan adds.
- Enter **writes immediately** (closes + emits a rename request) instead of
  advancing to the service checklist — rename does not touch membership, so
  there is no step 2.
- Uniqueness validation **excludes the current name**: renaming `core` to
  `core` must say "already named" rather than the misleading "already
  exists", and must not write a no-op edit (every write re-encodes the whole
  file, which closes blank lines — see README's documented YAML limitation —
  so a pointless rewrite is a real cost).
- No `serviceNames` parameter — the create modal carries them only for its
  step 2. The rename constructor is
  `GroupNameModalForRename(currentName, existingGroups) tea.Model`.

No confirm step. Rename is not destructive — it is a value rename, easily
reversed, unlike delete-group (which strips tags) or remove-containers.
`ConfirmModal` is reserved for those.

### Message flow

Identical to the create/edit group flow: the modal knows the names, AppModel
knows the loaded file.

```
GroupsList: R ──> cmds.OpenRenameGroupModal(group)
AppModel:   OpenRenameGroupModalMsg ──> activeModal = GroupNameModalForRename(...)
Modal:      Enter ──> cmds.CloseModal(cmds.RequestRenameGroup(old, new))
AppModel:   RenameGroupRequestMsg ──> cmds.RenameGroup(configFileName, old, new)
cmd:        RenameGroup ──> utils.RenameGroupTag(fileName, old, new)
AppModel:   RenameGroupMsg{Err, NewName} ──> error → reportForegroundError;
            success → set selection to NewName, reload config, re-sync lists
```

## Data-layer change: `utils.RenameGroupTag`

```go
// RenameGroupTag replaces every profiles entry equal to oldName with
// newName in the compose file at fileName. Other profiles and every other
// key are untouched. Nothing else in a compose file references a profile
// by name (unlike service names, which depends_on references — which is
// why service renames are refused), so a rename cannot leave dangling
// references. Returns an error when no service carries oldName, which
// catches the file having changed since the modal was opened.
func RenameGroupTag(fileName string, oldName string, newName string) error {
	doc, err := readComposeNode(fileName)
	if err != nil {
		return err
	}

	servicesNode, err := servicesMappingNode(doc)
	if err != nil {
		return err
	}

	renamed := false
	for i := 0; i+1 < len(servicesNode.Content); i += 2 {
		serviceNode := servicesNode.Content[i+1]
		profilesNode := findMappingValue(serviceNode, "profiles")
		if profilesNode == nil {
			continue
		}
		for _, item := range profilesNode.Content {
			if item.Value == oldName {
				item.Value = newName
				renamed = true
			}
		}
	}

	if !renamed {
		return fmt.Errorf("group %q not found in compose file", oldName)
	}

	return writeComposeNode(fileName, doc)
}
```

Properties, all consistent with the existing `AddGroupTag`/`RemoveGroupTag`/
`SetGroupMembers`:

- Single read-modify-write pass, atomic via `ReplaceFileAtomically`.
- Replaces **every** occurrence (a service carrying the tag more than once
  — possible only via hand-edited YAML, compose-go dedups on load — is
  fully renamed).
- Node-tree edit, so comments and formatting on untouched lines ride
  through (the same guarantee `TestSetGroupMembers_PreservesComments`
  pins for membership edits). The known blank-line compaction on write is
  shared with every group op and the inline editor.
- Scalar quoting: `encodeNode` uses the yaml.v3 encoder (indent 2), which
  quotes special characters — identical to the create path, so names like
  `core:prod` already work today and keep working.
- Deliberately **no** collision check and **no** dedup in the writer: the
  modal owns uniqueness (it has the group list); the writer does the
  mechanical walk. This mirrors how `AddGroupTag` does not check
  collisions either. A service carrying both old and new after a rename is
  only reachable through a race or a direct utils call, and duplicate
  profile entries are harmless (docker treats a profile as a set; the UI
  dedups via `utils.Deduplicate`).

## Detailed changes

### 1. `src/keys/Keys.go` — declare the binding

Add to `ListKeys`:

```go
// ListKeys act on the body's left panel... New, Edit and Delete only mean
// something on the groups list...
//   (extend the comment: Rename joins them)
type ListKeys struct {
	// ...
	Rename key.Binding
}
```

```go
var List = ListKeys{
	// ...
	Delete: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
	// R on the groups list renames the highlighted group. Uppercase so it
	// does not collide with the details panel's lowercase r (restart);
	// same shape as E next to e on the services panel.
	Rename: key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "rename")),
}
```

`keys.Active`, Home / list branch:

```go
own := []key.Binding{List.New}
if !ctx.ListEmpty {
	own = append(own, List.Edit, List.Delete, List.Rename)
}
```

`keys.Catalog`, List scope entries: add `List.Rename` to the
`entries(List.Select, List.New, List.Edit, List.Delete, ...)` call.

### 2. `src/cmds/RenameGroup.go` (new)

```go
package cmds

type RenameGroupMsg struct {
	Err error
	// NewName rides the result so AppModel can keep the renamed group
	// selected after the reload — selection.groupName still holds the old
	// name until then.
	NewName string
}

// RenameGroupRequestMsg asks AppModel to rename GroupName to NewName in
// the loaded compose file. The rename modal emits this instead of the
// command itself, the same split as CreateGroupRequestMsg.
type RenameGroupRequestMsg struct {
	GroupName string
	NewName   string
}

func RequestRenameGroup(groupName string, newName string) tea.Cmd {
	return func() tea.Msg {
		return RenameGroupRequestMsg{GroupName: groupName, NewName: newName}
	}
}

// RenameGroup renames groupName to newName in fileName, the compose file
// AppModel has loaded. See utils.RenameGroupTag.
func RenameGroup(fileName string, groupName string, newName string) tea.Cmd {
	return func() tea.Msg {
		return RenameGroupMsg{Err: utils.RenameGroupTag(fileName, groupName, newName), NewName: newName}
	}
}
```

### 3. `src/cmds/OpenRenameGroupModal.go` (new)

```go
package cmds

type OpenRenameGroupModalMsg struct {
	GroupName string
}

func OpenRenameGroupModal(groupName string) tea.Cmd {
	return func() tea.Msg { return OpenRenameGroupModalMsg{GroupName: groupName} }
}
```

### 4. `src/components/GroupNameModal.go` — rename mode

Add fields to `GroupNameModalModel`:

```go
type GroupNameModalModel struct {
	input          textinput.Model
	existingGroups []string
	serviceNames   []string
	errMsg         string
	// isRename switches step 1 of the create flow into the whole rename
	// flow: Enter writes instead of advancing to the service checklist.
	isRename    bool
	currentName string
	termHeight  int
}
```

Submit handler becomes:

```go
case key.Matches(keyMsg, keys.Overlay.Submit):
	name := m.input.Value()

	switch {
	case name == "":
		m.errMsg = "Group name can't be empty"
	case m.isRename && name == m.currentName:
		// Same name is a no-op that would still rewrite the whole file
		// (closing blank lines); say so rather than doing it.
		m.errMsg = fmt.Sprintf("Group is already named %q", name)
	case slices.Contains(m.existingGroups, name):
		m.errMsg = fmt.Sprintf("Group %q already exists", name)
	case m.isRename:
		return m, cmds.CloseModal(cmds.RequestRenameGroup(m.currentName, name))
	default:
		return ServiceChecklistModal(name, m.serviceNames, m.termHeight), nil
	}

	return m, nil
```

View: derive title and submit hint from `m.isRename` (`"Rename group"` /
`"rename"` vs `"New group"` / `"next"`). The create signature and behavior
are untouched, so the existing `modal_chrome_test.go` case and create-flow
tests stay green.

New constructor:

```go
// GroupNameModalForRename is the rename flow: prompt for the group's new
// name, pre-filled with the current one. Enter writes the rename (via
// RequestRenameGroup); Esc cancels. Uniqueness excludes the current name,
// so renaming core to core gets its own message rather than "already
// exists". No termHeight: unlike create, there is no step 2 checklist to
// size.
func GroupNameModalForRename(currentName string, existingGroups []string) tea.Model {
	input := textinput.New()
	input.Placeholder = "e.g. core"
	input.SetWidth(30)
	input.SetValue(currentName) // cursor lands at end; ctrl+u clears
	input.Focus()

	return GroupNameModalModel{
		input:          input,
		existingGroups: existingGroups,
		isRename:       true,
		currentName:    currentName,
	}
}
```

### 5. `src/components/GroupsList.go` — handle `R`

Next to the `Edit` case:

```go
case key.Matches(msg, keys.List.Rename):
	if selectedGroup, ok := m.list.SelectedItem().(apptypes.GroupListItem); ok {
		finalCmds = append(finalCmds, cmds.OpenRenameGroupModal(string(selectedGroup)))
	}
```

List-only, like `e`/`d` — a subject is required, so it never fires with an
empty or unfocused list, and `n` (the one verb that works from both panels)
stays in AppModel where it is.

### 6. `src/model/Update.go` — wire the messages

```go
case cmds.OpenRenameGroupModalMsg:
	if m.config.configProject != nil {
		m.activeModal = components.GroupNameModalForRename(
			msg.GroupName, m.allGroupNames(),
		)
	}

case cmds.RenameGroupRequestMsg:
	// Same split as CreateGroupRequestMsg: the modal knows the names,
	// AppModel knows the file the rename must be written into.
	finalCmds = append(finalCmds, cmds.RenameGroup(
		m.config.configFileName, msg.GroupName, msg.NewName,
	))

case cmds.RenameGroupMsg:
	m.lastErrorFromPoll = false
	if msg.Err != nil {
		finalCmds = append(finalCmds, m.reportForegroundError(msg.Err.Error()))
	} else {
		m.lastError = ""
		// Keep the renamed group selected: configSyncCmds re-selects
		// selection.groupName after the reload, and it still holds the
		// old name until now.
		m.selection.groupName = msg.NewName
		finalCmds = append(finalCmds, cmds.GetConfig(m.config.source))
	}
	if cfCmd := m.recomposeFilesCmdIfActive(); cfCmd != nil {
		finalCmds = append(finalCmds, cfCmd)
	}
	if bodyCmd := m.rebroadcastBodyLayoutIfChanged(); bodyCmd != nil {
		finalCmds = append(finalCmds, bodyCmd)
	}
```

The `RenameGroupMsg` handler is a copy of `CreateGroupMsg` plus the
one-line selection preservation. If the rename fails, `selection.groupName`
is untouched and the old name is still selected — correct.

## Edge cases

| Case | Behaviour |
|------|-----------|
| New name empty | Inline error, existing check |
| New name == current name | Inline error "already named" — no wasted rewrite (every write re-encodes the file and closes blank lines) |
| New name collides with another group | Inline error, existing check (minus self) |
| Case differences (`core` vs `Core`) | Distinct profiles; checks stay case-sensitive, matching docker |
| Old name absent from file at write time (file changed between list-sync and Enter) | `RenameGroupTag` returns an error → foreground error modal/banner, nothing written |
| Group tagged on services with *other* profiles | Other entries untouched (the walk only replaces `oldName`) |
| Special characters / quoting in names | yaml.v3 quotes on encode, same as create today |
| Rename vs pending docker action | Independent: docker already received its command; the next start uses the new name via `--profile newName up` |
| Rename vs running containers | Containers are per-service, not per-profile; unaffected. The details panel re-derives from the reloaded project |
| Selection after rename | Preserved via `m.selection.groupName = msg.NewName` before reload; on failure the old selection stands |
| Group becomes empty | Impossible — rename does not change membership |
| Profiles key becomes empty | Impossible — rename never removes the key (only delete does) |
| List filter active | `R` is a letter while typing (list owns the keyboard); after the filter is applied, `R` renames the item under the cursor, same as `e`/`d` |
| `R` on the services list / details panel | Unbound there; footer does not advertise it (only Home/list `Active()` includes it) |

## Known limitations (document, do not fix here)

- **Single-file scope.** The app loads exactly one compose file
  (`config.configFileName`). A rename rewrites that file only. DESIGN.md's
  "multi-file YAML rewriting" concern is about projects whose services span
  several files (`docker compose -f a.yml -f b.yml`); those other files are
  invisible to the app, so tags there keep the old name and the group would
  appear to re-appear on the next reload. This is inherent to how the app
  already loads files and how create/delete work; the plan documents it in
  DESIGN.md rather than solving it (solving it is a load-multiple-files
  project, out of scope).
- **External references.** `--profile <name>` in CI scripts or shell
  aliases is user space; the app cannot see or rewrite it. Same caveat as
  create/delete.
- **Read-modify-write race.** An external edit landing between the app's
  read and atomic write is clobbered — the pre-existing behaviour of every
  group op, unchanged here.

## Unknowns to confirm during implementation

1. **Nothing else in a compose file references a profile name.** This plan
   asserts it (which is why a value walk is safe where a service rename is
   refused). Confirm against the compose spec (`profiles` is per-service
   definition only; `depends_on` references services, not profiles) before
   finalizing the DESIGN.md wording. Low risk — the walk touches only
   `profiles:` sequences, so it is safe within the file regardless.
2. **bubbles v2 textinput behaviour on `SetValue`** — cursor lands at the
   end; `ctrl+a` is line-start, `ctrl+u` clears, no select-all. Verified in
   the vendored source (v2.1.0). The prefill UX assumes this.
3. **compose-go `DisabledServices`** — `allGroupNames()` reads only
   `configProject.Services`. `ReadConfigFile` loads with no profile filter,
   so every service (profile-tagged or not) lands in `Services`; verified by
   reading compose-go's load path and the existing `groupProject()` test
   fixture. The rename write itself reads raw YAML, so it is independent of
   this either way.
4. **Footer width** — one more hint in the Home/list footer. The
   narrow-terminal wrapping is an open TODO; the plan accepts the addition
   and notes the help-only fallback.

## Tests

Follow the house patterns (CONTRIBUTING.md): component and model tests first,
rig only for the full flow.

### `src/utils/GroupTags_test.go` — `RenameGroupTag`

Using the existing `baseFixture` / `multiTagFixture` / `writeFixture` /
`readServiceGroups` helpers:

- `TestRenameGroupTag_RetagsEveryCarrier` — rename `core` → `core2` over
  `baseFixture`; both `app` and `db` carry `core2`, `cache` stays untouched
  and gains no profiles key.
- `TestRenameGroupTag_PreservesOtherProfiles` — `multiTagFixture`, rename
  `core` → `core2`: `app` keeps `extra` alongside the renamed tag.
- `TestRenameGroupTag_PreservesComments` — mirror
  `TestSetGroupMembers_PreservesComments`: the `# core services` comment
  survives.
- `TestRenameGroupTag_NotFound` — renaming a name no service carries
  returns an error and leaves the file byte-identical.
- `TestRenameGroupTag_IdempotentForNewName` — renaming to a name that
  already exists elsewhere merges (deliberate writer behaviour; collision is
  the modal's job).

### `src/components/GroupNameModal_test.go` (new)

- Prefilled with the current name, title `Rename group`, Enter hint
  `rename`.
- Enter with the unchanged name → inline error, modal stays open, no
  request emitted.
- Enter with a name colliding with another group → inline error (the
  create-flow cases already cover the base validation).
- Enter with a valid new name → `CloseModalMsg` whose follow command emits
  `RenameGroupRequestMsg{GroupName, NewName}` — mirror the edit-membership
  test's two-step close handling.
- Esc → closes, no request.

### `src/model/rename_test.go` (new, mirrors `editgroup_test.go`)

Reuse `groupProject()` / `withGroupsLoaded()` helpers:

- `TestPressingROpensTheRenameModal` — `R` on the focused list opens a
  `GroupNameModalModel` in rename mode for the highlighted group.
- `TestRenameRequestIsEmittedOnEnter` — the modal emits
  `RenameGroupRequestMsg`, and *not* `CreateGroupRequestMsg`/
  `EditGroupRequestMsg`.
- `TestRenameRequestIsBoundToTheLoadedFile` — AppModel turns the request
  into a `RenameGroup` command carrying `configFileName`.
- `TestRenameSuccessReloadsConfigAndKeepsSelection` — success sets
  `m.selection.groupName` to the new name and emits `GetConfig`.
- `TestRenameFailureShowsErrorAndKeepsOldSelection` — `RenameGroupMsg{Err}`
  leaves `lastError` set and `selection.groupName` at the old name.

### Updated existing tests

- `src/keys/Keys_test.go` — `TestCatalogAvailability` "groups list with
  groups": add `List.Rename` to the available set; the empty-list subtest
  must *not* include it.
- `src/components/KeybindingBar_test.go` — `TestFooterHints` line 41 pins
  `"space start · n new · e edit · d delete · / filter · ↑/↓ navigate · tab
  next"`; becomes `"space start · n new · e edit · d delete · R rename ·
  / filter · ↑/↓ navigate · tab next"`. Check the other pinned rows for
  drift.
- `src/components/modal_chrome_test.go` — add a rename case to
  `TestEveryModalHasATitleAndAnExitHint`:
  `{"rename group", GroupNameModalForRename("core", nil), "Rename group", "esc"}`.
- `src/model/background_test.go` — `TestNoBackgroundBleedInModals`: add a
  rename-modal case (open modal while a poll is due; assert no
  `GetRunningContainersMsg`), matching the theme-picker case.

### Optional rig test

`TestRigRenameGroup` in the `rig_test.go` style: select a group, send `R`,
wait for `Rename group`, type a new name, Enter, wait for the new name in
the list. Timing-based like the other rig tests.

## Documentation updates

- `docs/DESIGN.md` §3 — replace "Renaming a group is currently not
  supported" with the implemented behaviour plus the single-file
  limitation (see *Known limitations*).
- `TODO.md` — mark the `[P] Group rename` item done, in the same voice as
  the `Edit group membership` entry.
- `README.md` — add `| R | Rename the highlighted group | Groups panel
  focused |` to the keybindings table; mention rename in the feature
  paragraph.
- `docs/ROADMAP.md` — drop "Group rename" from the post-alpha list.

## Implementation order

Each step compiles and passes tests on its own.

| Step | File | What | Risk |
|------|------|------|------|
| 1 | `src/utils/GroupTags.go` | `RenameGroupTag` + its tests | Isolated, no callers yet |
| 2 | `src/cmds/RenameGroup.go`, `src/cmds/OpenRenameGroupModal.go` | Message types + commands | New files |
| 3 | `src/components/GroupNameModal.go` | Rename mode + `GroupNameModalForRename` + component tests | Touches the create flow — existing create tests guard it |
| 4 | `src/components/GroupsList.go` | `R` handler | New branch beside `e`/`d` |
| 5 | `src/keys/Keys.go` | `List.Rename`, `Active()`, `Catalog()` | Breaks the pinned footer/catalog tests — fixed in the same step |
| 6 | `src/model/Update.go` | Three message cases | Mirrors `CreateGroupMsg` block |
| 7 | Tests | `rename_test.go`, `background_test.go`, `modal_chrome_test.go` | New + additions |
| 8 | Docs | DESIGN, TODO, README, ROADMAP | Prose only |
| 9 | Full pass | `go build ./... && go vet ./... && go test ./... && gofmt -l .` and `go test -race ./...` | CI gates |

## Acceptance criteria

1. `R` on a focused, non-empty groups list opens a modal titled
   `Rename group`, pre-filled with the current name, cursor at end.
2. Enter with the unchanged name shows "already named" inline; the modal
   stays open.
3. Enter with a name that collides with another group shows "already
   exists" inline; nothing is written.
4. Enter with a valid name rewrites every `profiles:` entry for the old
   name in the loaded file, preserves comments/other profiles, closes the
   modal, and reloads the app with the renamed group still selected.
5. Esc cancels; nothing is written.
6. A rename failure (e.g. file edited externally in between) surfaces in
   the foreground error path; the old selection stands.
7. `R` is advertised in the footer on the Home list with groups, and in the
   `?` overlay under the List scope; it appears nowhere on the Services
   list, details panels, or Files page.
8. `R` is a letter while a list filter is being typed.
9. All checks green: `go build`, `go vet`, `go test`, `gofmt -l`, and
   `go test -race`.
