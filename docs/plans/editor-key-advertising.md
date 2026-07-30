# Step 5: Advertise the Editor's Keys

Part of [PLAN-EDITOR-UX.md](../PLAN-EDITOR-UX.md). Depends on
[step 4](editor-indent-keys.md) for the `keys.Editor` scope.

## Problem

Two things, one cause.

**The new keys are invisible.** After step 4, tab and shift+tab do something
in the editor and nothing says so.

**Worse, the footer and help overlay actively lie about them.** While editing,
the help overlay's Global scope shows `tab next` and `shift+tab prev` as
available — but the editor owns the keyboard, so those keys indent instead.
The overlay's whole promise is that it reads the same bindings the handlers
match against and therefore cannot drift; this is a drift.

**And the editor's status line is the one place in the app that writes key
names by hand.** `renderStatusLine` (`src/components/DetailsPanel.go:796`):

```go
hints := renderKeyHints([]KeyHint{
	{Key: "ctrl+s", Desc: "save"},
	{Key: "ctrl+o", Desc: "editor"},
	{Key: "esc", Desc: "cancel"},
}, appstyles.Active.TextDim)
```

Those three strings duplicate `keys.Details.Save`, `keys.Details.OpenEditor`
and `keys.Global.Back`. Everywhere else in the codebase uses `hintFor` /
`hintAs`, which read `binding.Help()` — see `ServiceChecklistModal`,
`ThemePickerModal`, `LogsModal`, and `KeybindingBar` itself.

## Solution

Three edits, all mechanical, plus one judgement call about the Global scope.

### 1. The status line reads its bindings

```go
hints := renderKeyHints([]KeyHint{
	hintFor(keys.Details.Save),
	hintFor(keys.Details.OpenEditor),
	hintFor(keys.Editor.Indent),
	hintFor(keys.Editor.Outdent),
	hintAs(keys.Global.Back, "cancel"),
}, appstyles.Active.TextDim)
```

`hintAs` for Back because its own help text is "back", and in the editor it
cancels. That distinction is exactly what `hintAs` exists for.

Five hints may not fit a narrow details panel. `renderStatusLine` already
clamps with `Width`/`MaxWidth`, so a narrow terminal truncates rather than
wraps — check what that looks like at 80 columns before settling. If it
truncates badly, drop `ctrl+o` from the status line (it stays in the footer
and the overlay) rather than shortening the descriptions into codes.

### 2. The footer offers the editor's keys while editing

`src/keys/Keys.go`, `Active()`, the `ctx.Editing` branch (around line 349):

```go
if ctx.Editing {
	return []key.Binding{
		Details.Save, Details.OpenEditor,
		Editor.Indent, Editor.Outdent,
		Global.Back,
	}
}
```

Order: save first (the one that commits), then the buffer keys, then the way
out. Same shape as the existing line.

### 3. An Editor scope in the help overlay

`Catalog()` (around line 430), a new scope after "Details":

```go
{
	// Only reachable with the inline editor open, so every row here is
	// dimmed everywhere else - which is the overlay saying "these are the
	// editor's keys" without a sentence of prose.
	Title: "Editor",
	Entries: entries(
		Editor.NewLine, Editor.Indent, Editor.Outdent,
		Details.Save, Details.OpenEditor,
	),
},
```

`Details.Save` and `Details.OpenEditor` appear in both Details and Editor.
That is fine and already precedented — `Details.EditFile` appears in both
Details and Files. Consider whether to *move* Save and OpenEditor out of the
Details scope entirely now that an Editor scope exists; they are only live
while editing, so Editor is arguably their real home. Either choice is
defensible; if they move, update `TestCatalogAvailability` in
`src/keys/Keys_test.go` and `TestHelpOverlayRendersTheCatalog` in
`src/model/help_test.go`, which name the scopes explicitly.

### 4. Availability: make `pressableNow` tell the truth

`pressableNow` (around line 465) adds the always-live globals to whatever
`Active(ctx)` returns. Because `Global.NextPanel` / `PrevPanel` are in that
always-live set, the overlay shows `tab next` as available while editing. It
is not.

Fix: while `ctx.Editing`, tab and shift+tab belong to the editor. Exclude
`Global.NextPanel` and `Global.PrevPanel` from the always-live set when
`ctx.Editing` is true, so the Global scope dims them and the Editor scope
lights up instead.

Note `containsBinding`/`sameBinding` compare bindings — check how
(`TestSameBinding` in `Keys_test.go` documents it). If they compare by
keystroke rather than identity, `Editor.Indent` ("tab") and
`Global.NextPanel` ("tab") may be indistinguishable to it, and the two scopes
will light and dim together. If so, that is a real constraint, not a bug to
work around: report it and pick the honest option (probably: the Global rows
dim while editing, and accept that the comparison is by keystroke). **Do not
change `sameBinding`'s semantics to make this come out nicer** — other
availability logic depends on it.

## Do not

- **Do not** write any key name as a string literal. That is the defect this
  step exists to remove; adding another instance while fixing the first would
  be perverse.
- **Do not** add editor keys to the footer outside `ctx.Editing`. They do
  nothing there.

## Tests

`src/components/` — the status line:

1. **`TestEditorStatusLineNamesTheEditorKeys`** — render a `DetailsPanelModel`
   in edit mode and assert the stripped view contains `ctrl+s`, `tab`,
   `shift+tab` and `esc`. Build the expectations from
   `keys.X.Help().Key`, not from literals, so the test cannot drift either.

`src/keys/Keys_test.go`:

2. **`TestEditorKeysAreLiveOnlyWhileEditing`** — with `Context{Page: "Services",
   Focused: constants.COMPONENT_BODY_DETAILS, Editing: true}`, the Editor scope's
   entries are available; without `Editing`, they are dimmed.
3. **Extend `TestCatalogAvailability`** with an editing sub-test asserting the
   Global panel-switch rows are dimmed while editing (or documenting the
   keystroke-comparison constraint above, if that is what is found).

`src/model/help_test.go`:

4. **Extend `TestHelpOverlayRendersTheCatalog`** to expect the `Editor` scope
   title, so the overlay is asserted to have the section at all.

`src/model/keyboard_test.go`: the footer bar is asserted there
(`ansi.Strip(m.components.KeybindingBar.View().Content)`).

5. **`TestFooterOffersTheEditorKeysWhileEditing`** — enter edit mode via the
   `editingWeb` helper from [step 1](editor-paste.md) and assert the footer
   names indent and outdent, and does not offer the page digits.

## Verification

```
go build ./... && go vet ./... && go test ./...
```

Manual, at 80 columns and at 200: open the editor and read the status line and
the footer; press `?` while editing and confirm the Editor scope is lit and the
Global panel rows are dimmed.

## Commit

Branch `editor-key-advertising`, merged `--no-ff`:

```
Advertise the inline editor's keys from their bindings

The status line was the one place in the app that wrote key names as string
literals, and the help overlay claimed tab still switched panels while the
editor owned the keyboard.
```
