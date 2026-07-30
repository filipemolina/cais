# Plan: Inline Editor UX — Paste and YAML-Aware Editing

Two requests against the inline service editor (`e` on a focused details
panel): make paste work, and make Enter behave like a YAML editor's Enter.
This document is the investigation, the effort/gain call, and the plan.

## Investigation

### Paste is broken, and it is not the textarea's fault

`bubbles/v2/textarea` already implements both paste paths:

| Path | What the terminal sends | textarea handling |
|---|---|---|
| Terminal paste (ctrl+shift+v, cmd+v, middle-click) | bracketed paste → `tea.PasteMsg` | `textarea.go:1223` inserts the content |
| `ctrl+v` | an ordinary key press | matches `KeyMap.Paste`, returns the `Paste` cmd, which reads the system clipboard and comes back as an unexported `pasteMsg`, handled at `textarea.go:1319` |

Bracketed paste is on by default in bubbletea v2 (`View.DisableBracketedPasteMode`
is false), so the terminal *is* sending us the pasted text.

The break is in our own message routing. `DetailsPanelModel.Update` forwards
messages to the editor from exactly one branch — `case tea.KeyPressMsg`
(`DetailsPanel.go:161`). Everything else falls off the end of the switch and
is dropped. So:

- `tea.PasteMsg` never reaches the textarea. **The paste is silently discarded.**
- `ctrl+v` does reach the textarea, which correctly returns the `Paste` cmd —
  but the `pasteMsg` that cmd produces comes back through `Update` as a
  non-key message and is dropped too. The round trip completes and inserts
  nothing.

Verified empirically by driving the real model: entering edit mode with
`web:\n  image: nginx\n` and feeding `tea.PasteMsg{Content: "  ports:\n    - 80:80\n"}`
leaves the buffer byte-identical.

Note `AppModel` is not the problem — `UpdateInnerComponent` (`AppModel.go:206`)
already forwards every message to the active page's components. The fix is
local to `DetailsPanel`.

One caveat on `ctrl+v` specifically: it goes through `atotto/clipboard`
(already an indirect dep), which on Linux shells out to `xclip`/`xsel`/
`wl-copy`. It fails on a bare TTY and over plain SSH. Bracketed paste has no
such dependency and is what the user's ctrl+shift+v / cmd+v actually uses. So
bracketed paste is the path that matters; `ctrl+v` is a bonus that works
where a clipboard tool exists.

### Enter has no indentation behaviour at all

Also verified against the real model: park the cursor at the end of
`  image: nginx`, press Enter, type `x` → the buffer is
`web:\n  image: nginx\nx\n`. The `x` lands in column 0. Every continuation
line in a YAML fragment has to be re-indented by hand, which in a
two-space-per-level format is where most of the typing goes.

Tab in the editor currently does nothing (confirmed — it is not bound by the
textarea's keymap and the app's focus-cycling tab is blocked by
`keyboardOwned()`), so tab and shift+tab are free to take.

### What the textarea gives us to build on

Everything needed is exported: `Line()` (logical row), `Column()` (logical
column), `Value()`, `InsertString()`, `SetCursorColumn()`. Auto-indent is
therefore *our* logic on top of a public API — no fork, no reflection.

`insertRunesFromUserInput` also handles multi-line inserts correctly
(splits on `\n`, preserves each line's own leading spaces, splices the tail
of the current line) and its sanitizer replaces tab characters with spaces.
That last one is a genuine gift for YAML, where a literal tab is a hard
parse error: text pasted from a web page with tab indentation arrives legal.

### What is *not* worth doing

- **Syntax highlighting inside the editor.** `textarea` renders its own runes
  and offers no styled-content hook; you would have to fork it. `highlight.YAML`
  stays where it is, on the read-only compose file panel.
- **Structural YAML awareness** (knowing that `ports:` takes a sequence, that
  `image:` takes a scalar). That needs a schema-aware parse of an incomplete
  document on every keystroke. Large effort, and the wrong bet — the win here
  is mechanical typing, not completion.

## Effort / gain

| # | Feature | Effort | Gain | Verdict |
|---|---------|--------|------|---------|
| 1 | Paste (`tea.PasteMsg` + the `ctrl+v` round trip) | **~15 lines** | Unblocks the copy-from-the-internet workflow entirely. Today the feature is not slow, it is *absent*. | **Do it** |
| 2 | Enter keeps the current line's indent | ~25 lines | Every multi-line edit stops needing manual spacing | **Do it** |
| 3 | Enter after a line ending in `:` indents one level deeper | ~10 lines on top of #2 | Matches how you actually write a nested block | **Do it** |
| 4 | Enter inside a `- ` item aligns to the item's content column | ~10 lines on top of #2 | Keeps sequence items lined up | **Do it** |
| 5 | tab / shift+tab indent / outdent the current line | ~40 lines | The escape hatch for when the auto-indent guessed wrong. Without it the only fix is spacebar and backspace. | **Do it** |
| 6 | Backspace at the start of an indent eats a whole level | ~20 lines | Removes the "backspace twice per level" tax | **Do it** |
| 7 | Editor-scope keys registered in `keys` and shown in the status line / help | ~30 lines | The status line currently hardcodes its three hints as string literals rather than reading the bindings | **Do it** |
| 8 | Syntax highlighting in the editor | Fork `textarea` | Cosmetic | **No** |
| 9 | Schema-aware completion | Very large | Speculative | **No** |

The analysis is positive and lopsidedly so. #1 is a bug fix disguised as a
feature request and should not wait for the rest. #2–#6 are one coherent
piece of work — a pure string function plus a thin layer of cursor
bookkeeping — that stays entirely inside `DetailsPanel` and a new sibling
file. Nothing here touches the compose writer, the save path, or validation,
so the blast radius is one panel in one mode.

Total: roughly 150 lines of implementation plus tests.

## Plan

Ordered so each step is independently shippable and independently revertable.
One commit per step, on a feature branch, merged `--no-ff`.

Each step has its own document under `docs/plans/`, written to be handed to
someone (or something) that has not read this one. They carry the exact file
and line anchors, the code to write, the traps found while investigating, the
test list, and an explicit "do not" section:

| Step | Document | Depends on |
|---|---|---|
| 1 | [editor-paste.md](plans/editor-paste.md) | — |
| 2 | [editor-indent-policy.md](plans/editor-indent-policy.md) | — |
| 3 | [editor-enter-autoindent.md](plans/editor-enter-autoindent.md) | 2 |
| 4 | [editor-indent-keys.md](plans/editor-indent-keys.md) | 2 |
| 5 | [editor-key-advertising.md](plans/editor-key-advertising.md) | 4 |

Step 1 is a bug fix and shares nothing with the rest — ship it first and on
its own. Steps 2–4 are one branch, one commit each. Step 5 closes the loop by
making the footer and help overlay tell the truth about the new keys.

### Step 1 — Route paste to the editor

`src/components/DetailsPanel.go`

Add a `tea.PasteMsg` case that forwards to the editor while editing (and
ignores it otherwise), then re-runs `updateValidationError()` so the status
line reflects the pasted text immediately.

For the `ctrl+v` round trip, the message type is unexported, so it cannot be
named in a case. Forward *unrecognised* messages to the editor while editing:

```go
default:
    if m.editing {
        var cmd tea.Cmd
        m.editor, cmd = m.editor.Update(msg)
        finalCmds = append(finalCmds, cmd)
        m.updateValidationError()
    }
```

This is safe because every message the panel acts on already has its own
case above, and it makes the panel forward-compatible with any other
internal textarea message (clipboard errors, future cursor messages).

**Tests** (`src/model/inline_edit_test.go`): a `tea.PasteMsg` in edit mode
lands in the buffer, multi-line paste keeps its own indentation, a paste
that breaks YAML shows in the status line, and a `tea.PasteMsg` outside edit
mode changes nothing.

### Step 2 — An indent policy function

New file `src/components/yamlindent.go`, pure and standalone:

```go
// indentAfter returns the leading whitespace a new line should start with,
// given the line being split and the column the cursor split it at.
func indentAfter(line string, col int) string
```

Rules, in order:

1. Base = the leading spaces of `line` truncated at `col`.
2. If the text before the cursor ends in `:` (ignoring trailing spaces and a
   trailing `# comment`), add one level (two spaces).
3. Otherwise, if the line's first non-space content starts with `- `, base
   becomes the column of the text *after* the dash, so `  - name: web` +
   Enter aligns under `name`.
4. Splitting a line mid-text (`col` before the end) uses the base indent
   only — no extra level. Guessing deeper when the user is breaking a line in
   half is worse than doing nothing.

Two-space indent as a package constant. No config knob until someone asks:
two is the compose convention and the fragments we generate.

**Tests** (`src/components/yamlindent_test.go`): table-driven over the rules
above plus the edge cases — empty line, whitespace-only line, `col` 0, `col`
past the end, a line that is only a comment, `ports:` vs `image: nginx` vs
`- "8080:80"` vs `- name: web`.

### Step 3 — Wire Enter through the policy

In `handleEditKey`, intercept Enter before the textarea sees it:

```go
case msg.Code == tea.KeyEnter:
    indent := indentAfter(m.currentLine(), m.editor.Column())
    m.editor.InsertString("\n" + indent)
```

`currentLine()` is a small helper: split `Value()` on `\n` and index by
`Line()`, guarding a row out of range.

`handleEditKey` currently signals "not mine" by returning a nil cmd, and its
callers then forward the key to the textarea. Enter is handled but produces
no command, so that contract needs one more bit of return — a `handled bool`
— rather than being overloaded onto the nil cmd. Small, mechanical, and it
makes the existing three cases read better too.

**Tests**: Enter at the end of `  image: nginx` gives a two-space indent;
Enter at the end of `ports:` gives four; Enter after `  - name: web` aligns
to `name`; Enter mid-line splits without adding a level; the resulting
document still parses.

### Step 4 — tab / shift+tab, and backspace over an indent

Three new bindings in `keys.Details` (or a new `keys.Editor` scope, which
reads better and is where step 5 wants them): `IndentLine`, `OutdentLine`.
Backspace stays unbound — it is the textarea's own key, we just special-case
it when the cursor sits in leading whitespace.

- tab: insert one level at the start of the line's text, cursor moves with it.
- shift+tab: remove up to one level of leading whitespace, no-op at column 0.
- backspace with only spaces to the left of the cursor: delete back to the
  previous multiple of the indent width instead of one space.

All three operate on the current line via `Value()` / `SetValue()` /
`SetCursorColumn()`, so they are the same shape as step 3.

Note: `SetValue()` resets the cursor, so each of these must restore row and
column explicitly. Worth a comment — it is the kind of thing that looks
redundant and is not.

**Tests**: indent then outdent is a round trip; outdent at column 0 is a
no-op; backspace in text is unchanged; backspace in indent jumps a level;
the cursor lands where the tests say it does.

### Step 5 — Advertise the new keys

`renderStatusLine` currently hardcodes `{Key: "ctrl+s", Desc: "save"}` and
friends as literals — the one place in the app that writes key names by hand
instead of reading `Help()` off the binding. Fix that while adding tab and
shift+tab to the editing footer (`keys.go`'s `ctx.Editing` branch) and the
help overlay's catalog.

**Tests**: the editing status line names every editor key, built from the
bindings, and the help overlay lists them under an Editor scope.

## Risks

- **Enter interception changes a key everyone uses.** Mitigated by the policy
  function being pure and heavily table-tested, and by the rule that a
  mid-line split adds nothing.
- **A `default:` forwarding branch is a wide net.** It only fires while
  editing, and every message the panel handles is already cased above it.
- **`SetValue` cursor reset** is the likely source of any bug in step 4;
  called out above so the tests pin the cursor position, not just the text.

## Out of scope

Syntax highlighting in the editor, schema-aware completion, configurable
indent width, and YAML reformatting of the whole fragment on save.
