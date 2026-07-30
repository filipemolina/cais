# Step 2: The Indent Policy Function

Part of [PLAN-EDITOR-UX.md](../PLAN-EDITOR-UX.md). Pure logic, no wiring —
step 3 is what calls it. Doing it alone first is deliberate: the whole
correctness risk of the auto-indent feature lives in this one function, and
here it can be tested exhaustively without a model, a panel, or a terminal.

## Problem

Pressing Enter in the inline editor produces a line at column 0. In a
two-space-per-level format, that means every continuation line is re-indented
by hand, which is where most of the typing in a YAML edit goes.

## Solution

A new file `src/components/yamlindent.go` holding one exported-to-the-package
function and no state:

```go
// indentAfter returns the leading whitespace a line should start with when
// Enter splits `line` at `col`.
func indentAfter(line string, col int) string
```

Plus the indent width as a package constant:

```go
// yamlIndent is one level of YAML nesting. Two spaces is the compose
// convention and what the fragments the editor opens with already use, so it
// is a constant rather than a setting until someone asks for one.
const yamlIndent = "  "
```

### The rules, in order

Let `before = line[:col]` (clamped — see below).

1. **Base indent** — the leading spaces of `before`. If `before` is entirely
   whitespace, the base is `before` itself: a user who has pressed space four
   times and then Enter meant those four spaces.

2. **Deepen after a block opener** — if the meaningful text of `before` ends
   in `:`, return base + `yamlIndent`. "Meaningful" means: trailing spaces
   stripped, and a trailing `# comment` stripped first. So all three of these
   deepen:
   - `ports:`
   - `ports:   `
   - `ports: # the ones we expose`

3. **Align inside a sequence item** — otherwise, if the first non-space
   content of `before` starts with `- `, the base becomes the column of the
   text *after* the dash and its following spaces. So `  - name: web` + Enter
   gives a four-space indent, landing the cursor under `name`, which is where
   the next key of the same mapping belongs.

   A bare `-` with nothing after it (or `- ` with only spaces) is treated as
   its own content column: dash column + 2.

4. **Mid-line splits do not deepen** — if `col` is before the end of the
   line's meaningful text, apply rule 1 (and rule 3) but never rule 2. Rule 2
   asks "is the user opening a block?", and someone breaking a line in half is
   not. Guessing deeper there is worse than doing nothing.

   Concretely: `col` is "at the end" if `line[col:]` is empty or all spaces.

### Edge cases the function must survive

`col` is a cursor position from the textarea, so treat it as untrusted:

- `col < 0` → clamp to 0.
- `col > len(line)` → clamp to `len(line)`. Note the textarea's `Column()` is
  a **rune** index while Go string slicing is byte-based. Convert the line to
  `[]rune` and work in runes, or the function mis-slices the moment a
  fragment contains a non-ASCII character (a comment in Portuguese, an
  em-dash in a label). Do this even though YAML keys are usually ASCII —
  comments and values are not.
- Empty line → `""`.
- A line that is only a comment (`# note`) → base indent only. It must not
  deepen even if the comment happens to end in `:`; rule 2 strips the comment
  before looking at the last character, so `# ports:` does not deepen.
- Tabs in the line: cannot happen (the textarea's sanitizer replaces tabs
  with spaces on every insert path), but treat a tab as an ordinary
  whitespace rune in the base-indent scan rather than special-casing it.

### Deliberately not handled

- **Strings containing `:` or `#`** — `image: "nginx:alpine"` ends in `e`, so
  rule 2 does not fire; that is correct by accident and good enough. A value
  like `command: echo "hi #"` would have its quoted `#` stripped as a comment
  by rule 2's scan, so the line reads as ending in `"hi` — still not `:`, so
  still no deepen. The failure mode of getting this wrong is one wrong indent
  level that shift+tab (step 4) fixes in one keystroke, which is not worth a
  YAML tokenizer here.
- **Block scalars** (`|`, `>`) — inside a block scalar, indentation is content
  and the rules above are wrong in principle. In practice rule 1 keeps the
  existing indent, which is the right behaviour for continuing a block scalar
  anyway. Leave it.
- **Anything requiring a parse of the document.** This function looks at one
  line. That is the whole design: it cannot be wrong about the document
  because it never claims to know it.

## Tests

New file `src/components/yamlindent_test.go`, table-driven. One table, each
row a `{name, line, col, want}`. Name the rows after the behaviour, not the
input, so a failure reads as a broken promise.

Required rows:

| line | col | want | why |
|---|---|---|---|
| `"  image: nginx"` | end | `"  "` | rule 1, the common case |
| `"web:"` | end | `"  "` | rule 2 at the top level |
| `"  ports:"` | end | `"    "` | rule 2 nested |
| `"  ports:   "` | end | `"    "` | rule 2 past trailing spaces |
| `"  ports: # exposed"` | end | `"    "` | rule 2 past a comment |
| `"  # ports:"` | end | `"  "` | a comment never opens a block |
| `"  - \"8080:80\""` | end | `"  - "`'s content column → `"    "` | rule 3 |
| `"  - name: web"` | end | `"    "` | rule 3 wins over rule 2's colon? **No** — see below |
| `"  -"` | end | `"    "` | bare dash |
| `"    "` | 4 | `"    "` | whitespace-only line |
| `""` | 0 | `""` | empty |
| `"  image: nginx"` | 9 (mid-word) | `"  "` | rule 4, no deepen |
| `"web:"` | 2 (mid-word) | `""` | rule 4: base of `"we"` is empty |
| `"  image: nginx"` | -1 | `"  "` | clamped |
| `"  image: nginx"` | 999 | `"  "` | clamped |
| `"  # níveis:"` | end | `"  "` | non-ASCII does not shift the slice |

**Resolve the `- name: web` row deliberately, and write the answer into a
comment in the function.** Both rules 2 and 3 have a claim: the line ends in a
value (not `:`), so rule 2 does not fire, and rule 3 gives the dash's content
column — four spaces. That is the intended answer. But `  - name:` (a dash
item whose last token *is* a key opening a block) should deepen from the
content column, giving six. Add both rows:

| `"  - name: web"` | end | `"    "` | rule 3, aligns under `name` |
| `"  - name:"` | end | `"      "` | rule 3 sets the base, then rule 2 deepens it |

So the order is: compute the base (rule 1, then rule 3 overrides it), then
apply rule 2 on top. Implement it in that order and the table above falls out.

Also add one integration-flavoured row set: a small table asserting that
applying the function repeatedly to build a nested block produces something
`yaml.Unmarshal` accepts. It guards against a rule that is individually
plausible and collectively produces invalid YAML.

## Verification

```
go test ./src/components/ -run Indent -v
go build ./... && go vet ./... && go test ./...
```

No manual check — nothing is wired yet. That is the point of splitting this
step out.

## Commit

Branch `editor-indent` (steps 2–4 share it; commit per step), first commit:

```
Add the YAML indent policy the editor's Enter will use

One pure function over one line of text, so the whole correctness question
is answered by a table test rather than by driving a terminal.
```
