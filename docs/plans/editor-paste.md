# Step 1: Route Paste to the Inline Editor

Part of [PLAN-EDITOR-UX.md](../PLAN-EDITOR-UX.md). Independent of steps 2–5 —
do this one first and ship it on its own.

## Problem

Pasting into the inline service editor (`e` on a focused details panel)
inserts nothing. The text is silently discarded.

This is not a missing feature in `bubbles/v2/textarea`. The textarea handles
both paste paths already:

| Path | Arrives as | Handled at |
|---|---|---|
| Terminal paste (ctrl+shift+v, cmd+v, middle-click) | `tea.PasteMsg` | `textarea.go:1223` |
| `ctrl+v` | key press → matches `KeyMap.Paste` → returns the `Paste` cmd → an **unexported** `pasteMsg` | `textarea.go:1319` |

Bracketed paste is on by default in bubbletea v2, so the terminal is sending
us the text. `AppModel.UpdateInnerComponent` (`src/model/AppModel.go:206`)
forwards every message to the active page's components, so it reaches
`DetailsPanelModel.Update`.

The break is in `DetailsPanelModel.Update`. It forwards to the editor from
exactly one branch — `case tea.KeyPressMsg` (`src/components/DetailsPanel.go:162`).
Every other message falls off the end of the type switch and is dropped:

- `tea.PasteMsg` never reaches the textarea.
- `ctrl+v` *does* reach it and correctly returns the `Paste` cmd, but the
  `pasteMsg` that cmd produces comes back as a non-key message and is dropped
  too. A complete round trip that inserts nothing.

## Solution

Two additions to the type switch in `DetailsPanelModel.Update`
(`src/components/DetailsPanel.go`, the switch starting around line 66):

### 1. An explicit `tea.PasteMsg` case

Place it immediately before `case tea.KeyPressMsg:` (line 162) — paste is a
sibling of key input and reads best next to it.

```go
// A terminal paste arrives as its own message, not as key presses. It only
// means anything to the editor, and only while the editor is open: the
// panel's read-only mode has nothing to paste into.
case tea.PasteMsg:
	if m.editing {
		var editorCmd tea.Cmd
		m.editor, editorCmd = m.editor.Update(msg)
		finalCmds = append(finalCmds, editorCmd)
		m.updateValidationError()
	}
```

`updateValidationError()` matters: without it the status line keeps saying
"YAML ok" until the next keystroke, which is exactly the moment a user who
just pasted a block wants the truth.

### 2. A `default` branch for the unexported round trip

`textarea`'s `pasteMsg` is unexported, so it cannot be named in a case. Add a
`default` at the end of the same switch:

```go
// The textarea's clipboard round trip (ctrl+v -> Paste cmd -> an unexported
// pasteMsg) comes back as a message this switch cannot name, so anything
// unrecognised goes to the editor while it is open. Safe by construction:
// every message the panel acts on has its own case above, so nothing that
// reaches here was ever the panel's to handle.
default:
	if m.editing {
		var editorCmd tea.Cmd
		m.editor, editorCmd = m.editor.Update(msg)
		finalCmds = append(finalCmds, editorCmd)
		m.updateValidationError()
	}
```

Both branches share a body. Factor it into one small method rather than
duplicating it:

```go
// forwardToEditor passes a message to the open editor and re-validates. It
// is the path for everything the editor answers that is not a key press.
func (m DetailsPanelModel) forwardToEditor(msg tea.Msg) (DetailsPanelModel, tea.Cmd) {
	if !m.editing {
		return m, nil
	}

	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	m.updateValidationError()

	return m, cmd
}
```

Note `updateValidationError` has a pointer receiver and `forwardToEditor` a
value receiver; that works (Go takes the address of the local copy), and
returning `m` keeps the value-receiver style the rest of the file uses.

## Do not

- **Do not** add a `ctrl+v` binding of our own. The textarea already has one,
  and duplicating it would mean two code paths for one key.
- **Do not** disable bracketed paste anywhere (`View.DisableBracketedPasteMode`).
- **Do not** try to make `ctrl+v` work without a clipboard tool. It goes
  through `atotto/clipboard` (already an indirect dependency), which on Linux
  shells out to `xclip`/`xsel`/`wl-copy` and fails on a bare TTY or plain SSH.
  That is fine and out of scope: bracketed paste needs no helper and is what
  ctrl+shift+v / cmd+v actually use. `ctrl+v` is a bonus where a clipboard
  tool exists.

## Tests

Add to `src/model/inline_edit_test.go` (model-level, so the whole routing
path is exercised — that is where the bug lives; a component-level test would
have passed all along).

There is no shared helper yet for "get into edit mode"; `TestInlineEditingOwnsTheKeyboard`
(line 179) does it inline. Extract that setup into a helper the new tests
share:

```go
// editingWeb puts the app in inline edit mode on the web service with the
// given fragment loaded, which is the starting state for the editor tests.
func editingWeb(t *testing.T, fragment string) AppModel
```

It should do what `TestInlineEditingOwnsTheKeyboard` does: `inlineEditProject(t)`,
`SetActivePageMsg("Services")`, `ChangeFocus(&details)` with
`constants.COMPONENT_BODY_DETAILS`, `SetSelectedServiceMsg` for `web`, then
`InlineEditReadyMsg{ServiceName: "web", Fragment: []byte(fragment)}`. Refactor
that existing test to use it.

Reading the buffer back needs an accessor. `DetailsPanelModel.editor` is
unexported and the test lives in `package model`, so add an exported reader on
the component:

```go
// EditorValue returns the editor's current contents. Exported for the model
// tests, which drive paste and indentation through the whole message path and
// need to see what landed in the buffer.
func (m DetailsPanelModel) EditorValue() string {
	return m.editor.Value()
}
```

Plus a test-side helper to find the panel in `m.pages["Services"]` and call it.
Steps 3 and 4 both need this too, so put it somewhere shared in the test files.

Cases:

1. **`TestPasteLandsInTheEditor`** — `tea.PasteMsg{Content: "  ports:\n    - \"8080:80\"\n"}`
   in edit mode; the buffer contains the pasted text.
2. **`TestPasteKeepsItsOwnIndentation`** — paste a two-line block with leading
   spaces; both lines keep their own indent. (This is the real workflow: a
   config copied off a web page.)
3. **`TestPasteRevalidates`** — paste something that breaks the YAML (e.g.
   `"\n  - [\n"`); assert the panel's status line reports a YAML error without
   any further keystroke. Read it via `View()` and `ansi.Strip`, matching how
   `bootstrap_test.go:187` checks for editor messages.
4. **`TestPasteOutsideEditModeIsInert`** — a `tea.PasteMsg` with the panel
   focused but not editing changes nothing and does not panic.

## Verification

```
go build ./... && go vet ./... && go test ./...
```

Manual check, since the whole point is a real terminal: run the app, `2` for
Services, focus the details panel, `e`, then ctrl+shift+v (or cmd+v) with a
service block on the clipboard. It should appear, and the status line should
re-evaluate.

## Commit

Branch `editor-paste`, one commit, merged `--no-ff`:

```
Route pasted text to the inline editor

The textarea already handles both paste paths; DetailsPanel only ever
forwarded tea.KeyPressMsg to it, so bracketed paste was dropped and the
ctrl+v clipboard round trip came back to a switch that had no case for it.
```
