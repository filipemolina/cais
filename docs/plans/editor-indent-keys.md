# Step 4: tab / shift+tab / backspace Over Indentation

Part of [PLAN-EDITOR-UX.md](../PLAN-EDITOR-UX.md). Depends on
[step 2](editor-indent-policy.md) for `yamlIndent`, and reads best after
[step 3](editor-enter-autoindent.md) (it uses the same `handled` contract).

## Problem

Auto-indent guesses. When it guesses wrong the only repair is the spacebar and
backspace, one column at a time. And there is no way to indent an existing
line at all: **tab currently does nothing in the editor** (verified — the
textarea's keymap does not bind it, and the app's own tab is stood down while
the editor owns the keyboard).

## Solution

Three keys, all operating on the current logical line.

### The key collision, and why it is fine

`keys.Global.NextPanel` is **tab** and `keys.Global.PrevPanel` is
**shift+tab** (`src/keys/Keys.go:132-133`). They are globally live — but
`AppModel.Update` gives up its own key handling entirely while a component
owns the keyboard (`src/model/Update.go:382`, `if m.keyboardOwned() { break }`),
and `DetailsPanelModel.OwnsKeyboard()` is true while editing. So tab and
shift+tab never reach the panel-switching code from inside the editor. They
are genuinely free.

This is also why step 5 matters: the footer and help overlay must stop saying
tab means "next panel" while editing, or they will be lying.

Verified that `key.NewBinding(key.WithKeys("shift+tab"))` matches
`tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}` and that a plain `"tab"`
binding does **not** cross-match it. So both are ordinary bindings; no `Mod`
inspection needed in the component.

### New bindings

In `src/keys/Keys.go`, a new scope struct beside the existing ones:

```go
// EditorKeys act inside the inline YAML editor, and only there. The editor
// owns the whole keyboard while it is open (see DetailsPanelModel.OwnsKeyboard),
// which is what makes tab and shift+tab available here at all - they are the
// panel-switching keys everywhere else, and the app stands down from them
// while the editor holds the keyboard.
type EditorKeys struct {
	NewLine key.Binding
	Indent  key.Binding
	Outdent key.Binding
}

var Editor = EditorKeys{
	// Matched by the editor's own handler, which indents the new line rather
	// than letting the textarea insert a bare newline.
	NewLine: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "new line")),
	Indent:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "indent")),
	Outdent: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "outdent")),
}
```

Backspace stays **unbound**. It is the textarea's own key and we only
special-case it when the cursor sits in leading whitespace; declaring a
binding would imply we own it everywhere, which we do not.

### The three behaviours

All three go in `handleEditKey` as `handled: true` cases, alongside step 3's
Enter case.

**tab — indent the current line one level.** Insert `yamlIndent` at the start
of the line's text (column 0 of the line, before its existing indent), and
move the cursor right by the same width so the text under it does not shift
away.

**shift+tab — outdent one level.** Remove up to `len(yamlIndent)` leading
whitespace characters from the current line. Removing *up to* one level, not
exactly one, matters: a line indented three spaces outdents to one, not to a
negative. At column 0 with no leading whitespace it is a no-op — and a no-op
means "do nothing", not "insert something else".

**backspace inside leading whitespace — delete back one level.** If everything
to the left of the cursor on this line is spaces and there is at least one,
delete back to the previous multiple of `len(yamlIndent)`; otherwise return
`handled: false` and let the textarea have its normal backspace. Guard both
edges: at column 0 it is not ours (the textarea merges with the line above,
which is correct), and in text it is not ours either.

### The cursor trap — read this before implementing

The textarea has no "replace the current line" API, so these three operate by
rebuilding the value: `Value()` → split → edit the row → join →
`SetValue(...)`.

**`SetValue` resets the cursor to the end of the buffer.** Every one of these
must restore the row and column explicitly afterwards:

```go
// SetValue moves the cursor to the end of the buffer, so the row and column
// have to be restored by hand. This looks redundant and is not: without it,
// indenting a line in the middle of a fragment throws the cursor to the
// bottom of it.
func (m *DetailsPanelModel) replaceLine(row int, text string, col int) {
	lines := strings.Split(m.editor.Value(), "\n")
	if row < 0 || row >= len(lines) {
		return
	}
	lines[row] = text

	m.editor.SetValue(strings.Join(lines, "\n"))

	// Walk the cursor back: SetValue leaves it at the end.
	m.editor.MoveToBegin()
	for i := 0; i < row; i++ {
		m.editor.CursorDown()
	}
	m.editor.SetCursorColumn(col)
}
```

`MoveToBegin`, `CursorDown` and `SetCursorColumn` are all exported
(`textarea.go:1082`, `:712`, `:723`). If a direct row setter turns out to
exist in the version in `go.mod`, use it — but do not reach into unexported
fields, and do not use `CursorDown` in a loop without the `MoveToBegin` reset
first (its clamping behaviour depends on where it starts).

Alternative worth trying first, and preferring if it works: for **tab** the
edit is a pure insertion at a known position, so `SetCursorColumn(0)` followed
by `InsertString(yamlIndent)` avoids `SetValue` entirely and the cursor
bookkeeping with it. Same for backspace-over-indent if the textarea exposes a
usable delete. Use the rebuild path only where insertion cannot express the
edit (outdent, which deletes).

Whichever path is taken, the tests below pin the cursor position, not just the
text. That is where the bug will be.

## Do not

- **Do not** make tab insert a literal tab character. Tabs are a hard parse
  error in YAML. (The textarea's own sanitizer replaces tab characters with
  four spaces on insert, so a pasted tab is already safe — but do not rely on
  that for a key we control.)
- **Do not** implement multi-line / selection indent. The textarea has no
  selection model; adding one is a much larger piece of work.
- **Do not** make backspace at column 0 do anything new.

## Tests

Component-level is fine for the line arithmetic, but prefer model-level in
`src/model/inline_edit_test.go` for consistency with steps 1 and 3, using
`editingWeb` and `EditorValue`.

Cases:

1. **`TestTabIndentsTheCurrentLine`** — cursor mid-line; tab adds two spaces at
   the start; the text under the cursor is unchanged and the cursor still sits
   on the same character (assert the column moved by 2).
2. **`TestShiftTabOutdents`** — a four-space line outdents to two.
3. **`TestIndentThenOutdentIsARoundTrip`** — tab then shift+tab returns the
   buffer to byte-identical, cursor included.
4. **`TestOutdentAtColumnZeroIsANoOp`** — an unindented line is unchanged, and
   nothing panics.
5. **`TestOutdentOfAPartialIndentClampsToZero`** — a three-space line outdents
   to one, not to a negative.
6. **`TestBackspaceInIndentEatsALevel`** — cursor at column 4 of a
   four-space-indented line; one backspace leaves column 2.
7. **`TestBackspaceInTextIsUnchanged`** — cursor after a word; backspace
   deletes exactly one character (regression guard on the `handled: false`
   fall-through).
8. **`TestBackspaceAtColumnZeroStillMergesLines`** — the textarea's own
   behaviour survives.
9. **`TestIndentKeysDoNotSwitchPanelsWhileEditing`** — tab in edit mode
   produces no `SetFocusMsg` / focus change. This is the collision guard; it
   is the test that fails if someone later moves the `keyboardOwned()` check.
10. **`TestTabStillSwitchesPanelsOutsideTheEditor`** — the same key, not
    editing, still changes focus. Pair with 9; together they pin the seam.

## Verification

```
go build ./... && go vet ./... && go test ./...
```

Manual: `e` on a service, then tab and shift+tab on various lines, backspace
from inside an indent, and tab with the editor closed (must still move
panels). Save and confirm the file matches.

## Commit

Third commit on branch `editor-indent`:

```
Add tab, shift+tab and indent-aware backspace to the inline editor

tab and shift+tab are the panel-switching keys everywhere else; the app
stands down from its own keys while the editor owns the keyboard, which is
what makes them free here.
```
