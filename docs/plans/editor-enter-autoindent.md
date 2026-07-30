# Step 3: Wire Enter Through the Indent Policy

Part of [PLAN-EDITOR-UX.md](../PLAN-EDITOR-UX.md). Depends on
[step 2](editor-indent-policy.md) (`indentAfter` must exist).

## Problem

Step 2 built the policy. Nothing calls it. Enter still produces a line at
column 0.

## Solution

Intercept Enter in the editor's key handler, insert `"\n" + indentAfter(...)`
ourselves, and do not let the textarea see the key.

### The `handleEditKey` contract has to change first

`handleEditKey` (`src/components/DetailsPanel.go:202`) currently signals "this
key was not mine" by returning a nil command, and its caller
(`DetailsPanel.go:165`) reads that as "forward the key to the textarea":

```go
if m.editing {
	updated, cmd := m.handleEditKey(msg)
	if cmd != nil {
		return updated, cmd
	}
	m = updated
	// Not a special edit key: pass it to the textarea and validate.
	...
}
```

Enter breaks that: it *is* handled, and it produces no command. Overloading
the nil cmd cannot express it. Add an explicit `handled bool` return:

```go
// handleEditKey answers the keys the editor owns: the control keys (save,
// open in $EDITOR, cancel) and the ones that edit the buffer through the
// indent policy rather than as plain text. handled reports whether the key
// was the editor's; when it is false the caller passes the key to the
// textarea as ordinary input.
//
// handled is a separate return rather than "cmd != nil" because Enter is
// handled and produces no command - the buffer edit happens here, in place.
func (m DetailsPanelModel) handleEditKey(msg tea.KeyPressMsg) (DetailsPanelModel, tea.Cmd, bool) {
```

Every existing case returns `true`; the fall-through returns `false`. Update
the caller:

```go
if m.editing {
	updated, cmd, handled := m.handleEditKey(msg)
	m = updated
	if handled {
		m.updateValidationError()
		if cmd != nil {
			finalCmds = append(finalCmds, cmd)
		}
		break
	}
	// ...existing forward-to-textarea path
}
```

Two behaviour notes on that rewrite:

- The old code did `return updated, cmd` — an early return that skipped the
  rest of `Update`. The new code should `break` out of the case and fall
  through to the bottom of the function instead, so the batched commands are
  returned the same way every other branch does. Check that nothing depended
  on the early return; the save/cancel paths only produce commands, so they
  do not.
- `updateValidationError()` after a handled key is new and wanted: an Enter
  that changes the buffer should refresh the status line, exactly as ordinary
  typing does.

### The Enter case

Add to `handleEditKey`, **before** the `keys.Global.Back` case (order does not
functionally matter — esc and enter are different keys — but reading the
buffer-editing keys together does):

```go
case key.Matches(msg, keys.Editor.NewLine):
	m.editor.InsertString("\n" + indentAfter(m.currentLine(), m.editor.Column()))
	return m, nil, true
```

`keys.Editor.NewLine` is declared in [step 4](editor-indent-keys.md), which
introduces the `Editor` scope. If step 4 has not landed yet, either do its
`keys` declaration first (preferred — it is three lines) or match
`msg.Code == tea.KeyEnter` and leave a `TODO` pointing at step 4. Do **not**
write `key.NewBinding(key.WithKeys("enter"))` inline in the component: this
codebase declares every binding in `src/keys/Keys.go` so the help overlay
cannot drift from the handlers.

`InsertString` splices at the cursor and moves it to the end of what was
inserted, so the cursor lands after the indent. That is the whole behaviour —
no `SetValue`, no cursor bookkeeping.

### The `currentLine` helper

```go
// currentLine is the logical line the cursor is on. The textarea soft-wraps,
// so this is the row in the value, not the row on screen.
func (m DetailsPanelModel) currentLine() string {
	lines := strings.Split(m.editor.Value(), "\n")
	row := m.editor.Line()
	if row < 0 || row >= len(lines) {
		return ""
	}
	return lines[row]
}
```

`Line()` returns the logical row and `Column()` the logical column (verified
against `textarea.go:631` and `:636`), so soft wrapping does not enter into
it. The bounds guard is not defensive noise — `Value()` trims a trailing
newline, so on a buffer ending in `\n` with the cursor on the last (empty)
row, `Line()` can equal `len(lines)`.

`strings` is already imported in `DetailsPanel.go`.

## Do not

- **Do not** touch the textarea's `KeyMap`. Removing its Enter binding would
  also change what `InsertString("\n")` does elsewhere and makes the editor's
  behaviour depend on two places instead of one.
- **Do not** re-indent existing lines, reformat the fragment, or strip
  trailing whitespace. Enter affects the line it creates and nothing else.
  Anything that rewrites text the user did not just type belongs in a
  separate, opt-in feature.

## Tests

Add to `src/model/inline_edit_test.go`, using the `editingWeb` helper and
`EditorValue` accessor introduced in [step 1](editor-paste.md).

Getting the cursor where a test needs it: after `InlineEditReadyMsg` the
cursor is at the end of the buffer. Drive it with real key presses
(`tea.KeyPressMsg{Code: tea.KeyUp}`, `{Code: tea.KeyEnd}`,
`{Code: tea.KeyHome}`) rather than reaching into the textarea — the point of a
model-level test is that the keys do it.

Cases:

1. **`TestEnterKeepsTheCurrentIndent`** — fragment `"web:\n  image: nginx\n"`,
   cursor at the end of the `image` line, Enter, type `x` → buffer contains
   `"  image: nginx\n  x"`.
2. **`TestEnterDeepensAfterABlockOpener`** — fragment `"web:\n  ports:\n"`,
   cursor at the end of `  ports:`, Enter, type `-` → the `-` sits at column 4.
3. **`TestEnterAlignsInsideASequenceItem`** — fragment
   `"web:\n  environment:\n    - name: web\n"`, cursor at the end of the item,
   Enter, type `v` → the `v` sits under `name`.
4. **`TestEnterMidLineDoesNotDeepen`** — cursor inside `  image: nginx`
   (before the value), Enter → the tail moves to a line with the base indent
   and no extra level, and nothing is lost from the buffer.
5. **`TestEnterKeepsTheDocumentParseable`** — build a small nested block using
   only Enter and typed text, then assert `yaml.Unmarshal` on
   `EditorValue()` succeeds and the panel's status line says YAML ok.
6. **`TestEnterOutsideEditModeIsUnchanged`** — Enter with the details panel
   focused but not editing must still do whatever it does today (it is
   `List.Select`'s alias, so this is a regression guard on not having stolen
   it globally). Assert against the existing behaviour, do not change it.

Also re-run the existing editor tests: `TestInlineEditingOwnsTheKeyboard`,
`editor_test.go`, `editor_e2e_test.go`, `esc_test.go`. The `handleEditKey`
signature change touches the path all of them go through.

## Verification

```
go build ./... && go vet ./... && go test ./...
```

Manual: `e` on a service, then Enter at the end of `image:` (should indent one
level in), at the end of a key with no value (should indent two), and in the
middle of a line (should not deepen). Then ctrl+s and confirm the file on disk
is what the editor showed.

## Commit

Second commit on branch `editor-indent`:

```
Indent the line Enter creates in the inline editor

handleEditKey grows an explicit handled return: Enter is a key the editor
owns that produces no command, which "cmd != nil" could not express.
```
