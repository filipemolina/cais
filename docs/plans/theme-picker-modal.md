# Theme Picker Modal — Implementation Plan

## Problem

Cais ships three themes (`cais-dark`, `cais-dusk`, and `cais-day`)
has no UI to switch between them. `appstyles.Active` is the documented seam
(docs/DESIGN.md, ROADMAP.md post-alpha list), and the registry
(`appstyles.Themes`) is already populated. The only missing piece is a way
for the user to pick.

## Solution

`T` opens a theme picker modal. The list shows every registered theme. As the
cursor moves, `appstyles.Active` is reassigned — the entire TUI repaints
live. `Esc` or `Enter` closes the modal, keeping whatever theme is active.
There is no "cancel" — the last theme previewed is the one that sticks.

Persistence is not implemented yet (a future `~/.config/cais/
config.yaml` will own it), but the architecture is designed so that adding
persistence later is a one-line change to a single function.

### Why there is no revert on Esc

Live preview means the user sees the theme before committing. There is no
"before" state to revert to — each cursor movement already changed
`appstyles.Active`. Closing the modal with either key is the same action:
dismiss the picker, leave `appstyles.Active` at whatever it was last set to.

### Persistence seam

All theme changes flow through one function:

```go
// appstyles/Theme.go

// setTheme assigns a new active theme. Everything that draws reads Active
// fresh, so the next frame repaints. Returns nil today; when config file
// persistence lands, it returns a command that writes the name to
// ~/.config/cais/config.yaml.
func setTheme(name string) tea.Cmd {
    if t, ok := Themes[name]; ok {
        Active = t
    }
    return nil
}
```

The modal calls `setTheme` on cursor move (live preview) and on close (save).
Today it returns `nil`. When config persistence lands, the function gains a
`cmds.SaveThemeToConfig(name)` return and every call site is already correct.
No refactor needed.

## Scope and risk

**Low risk, additive.** One new component, one new keybinding, three new
message types, one new case in `AppModel.Update`. No existing components
change. No keybinding semantics change. The footer stays as-is.

### What does NOT change

| Aspect | Status |
|--------|--------|
| Groups list, Services list, Files page | **Unchanged** |
| Footer (`KeybindingBar`) | **Unchanged** — `T` is not in `Globals()`, no room |
| Help overlay catalog structure | **Unchanged** — one new row in the Global scope |
| `keys.Active()` | **Unchanged** — `T` is a global like `a`, handled in `pressableNow` only |
| `keys.Globals()` | **Unchanged** — same reason |
| `appstyles.Themes` registry | **Unchanged** — both existing themes stay |
| Background-bleed tests | **Unchanged** — the modal uses `ModalBg` like every other modal |

## Detailed changes

### 1. `src/appstyles/Theme.go` — add `setTheme()`

```go
// setTheme assigns a new active theme. Everything that draws reads Active
// fresh on each render (see Theme doc comment), so the next frame repaints
// in the new palette. Returns nil today; when config file persistence
// lands, this function grows a command that writes the name to disk.
func setTheme(name string) tea.Cmd {
    if t, ok := Themes[name]; ok {
        Active = t
    }
    return nil
}
```

This is an unexported package function. The modal calls it; no other call site
needs to change. When the config file work lands:

```go
func setTheme(name string) tea.Cmd {
    if t, ok := Themes[name]; ok {
        Active = t
    }
    return cmds.SaveConfigValue("theme", name)  // one line added
}
```

### 2. `src/keys/Keys.go` — add `Theme` binding

#### 2a. New field on `GlobalKeys`

```go
type GlobalKeys struct {
    // … existing fields …

    // Theme opens the theme picker modal. Like About, it is a global that
    // does not appear in the footer — the footer is width-constrained and
    // the help overlay is the comprehensive list.
    Theme key.Binding
}
```

#### 2b. New binding in the `Global` var

```go
var Global = GlobalKeys{
    // … existing bindings …
    Theme: key.NewBinding(key.WithKeys("T"), key.WithHelp("T", "theme")),
}
```

`T` is not used by any panel, list, or overlay — verified by grep.

#### 2c. Add to `pressableNow()`

Follow the same pattern as `About` — a global that appears in the help
overlay but not the footer:

```go
func pressableNow(ctx Context) []key.Binding {
    live := append(Active(ctx), Globals()...)
    live = append(live, Global.ForceQuit, Global.PrevPage, Global.NextPage,
        Global.About, Global.Theme)
    // … rest unchanged
}
```

#### 2d. Add to the Global scope in `Catalog()`

```go
{
    Title: "Global",
    Entries: entries(
        Global.NextPanel, Global.PrevPanel, Global.Back,
        Global.Quit, Global.ForceQuit, Global.Help, Global.About,
        Global.Theme,
    ),
},
```

### 3. `src/cmds/OpenThemePickerModal.go` (new file)

```go
package cmds

import tea "charm.land/bubbletea/v2"

type OpenThemePickerModalMsg struct{}

func OpenThemePickerModal() tea.Cmd {
    return func() tea.Msg { return OpenThemePickerModalMsg{} }
}
```

### 4. `src/cmds/ThemeChangedMsg.go` (new file)

```go
package cmds

import tea "charm.land/bubbletea/v2"

// ThemeChangedMsg tells AppModel that the theme picker closed and the
// active theme may have changed. Today AppModel does nothing with this —
// the theme was already applied live by the modal. When config file
// persistence lands, AppModel handles this message by calling
// SaveConfigValue to write the active theme name to disk.
type ThemeChangedMsg struct {
    Name string
}

func ThemeChanged(name string) tea.Cmd {
    return func() tea.Msg { return ThemeChangedMsg{Name: name} }
}
```

### 5. `src/components/ThemePickerModal.go` (new file)

```go
package components

import (
    "fmt"
    "io"

    "charm.land/bubbles/v2/key"
    "charm.land/bubbles/v2/list"
    tea "charm.land/bubbletea/v2"
    "charm.land/lipgloss/v2"
    "github.com/filipemolina/cais/src/appstyles"
    "github.com/filipemolina/cais/src/apptypes"
    "github.com/filipemolina/cais/src/cmds"
    "github.com/filipemolina/cais/src/keys"
)

// themeItem is a list row in the theme picker. It implements list.Item.
type themeItem struct {
    name string
}

func (t themeItem) Title() string       { return t.name }
func (t themeItem) FilterValue() string { return t.name }

// themePickerDelegate renders one theme row. The active theme gets an
// accent-colored left border so the user sees which one is current before
// they even move the cursor.
type themePickerDelegate struct {
    currentTheme string
}

func (d themePickerDelegate) Height() int  { return 2 }
func (d themePickerDelegate) Spacing() int { return 0 }
func (d themePickerDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
    return nil
}

func (d themePickerDelegate) Render(
    w io.Writer, m list.Model, index int, item list.Item,
) {
    ti, ok := item.(themeItem)
    if !ok {
        return
    }

    isSelected := index == m.Index()
    isActive := ti.name == d.currentTheme

    style := lipgloss.NewStyle().
        Width(m.Width()).
        Padding(0, 1).
        Background(appstyles.Active.ModalBg)

    if isActive {
        style = style.
            BorderLeft(true).
            BorderStyle(lipgloss.ThickBorder()).
            BorderLeftForeground(appstyles.Active.Accent)
    }

    titleStyle := lipgloss.NewStyle().
        Bold(isSelected).
        Background(appstyles.Active.ModalBg)

    if isSelected {
        titleStyle = titleStyle.Foreground(appstyles.Active.TextPrimary)
    } else {
        titleStyle = titleStyle.Foreground(appstyles.Active.TextMuted)
    }

    label := ti.name
    if isActive {
        label += " ●"
    }

    fmt.Fprint(w, style.Render(titleStyle.Render(label)))
}

// ThemePickerModalModel is the theme picker. It applies themes live as the
// cursor moves (assigning appstyles.Active via appstyles.setTheme), and
// emits cmds.ThemeChanged on close so the app can persist the choice later.
type ThemePickerModalModel struct {
    list         list.Model
    originalName string // theme that was active when the modal opened
}

func (m ThemePickerModalModel) Init() tea.Cmd { return nil }

func (m ThemePickerModalModel) Update(
    msg tea.Msg,
) (tea.Model, tea.Cmd) {
    if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
        switch {
        case key.Matches(keyMsg, keys.Overlay.Cancel),
            key.Matches(keyMsg, keys.Overlay.Submit):
            // Both keys close. The theme was already applied live;
            // emit ThemeChanged so the app can persist it later.
            name := appstyles.Active.Name
            return m, cmds.CloseModal(cmds.ThemeChanged(name))
        }
    }

    // Run the inner list update — this processes cursor movement.
    var listCmd tea.Cmd
    m.list, listCmd = m.list.Update(msg)

    // After the list update, check if the cursor moved to a different
    // theme. If so, apply it live. This is what makes the preview
    // instant: the frame after the cursor moves is the frame the
    // palette changes.
    if selected, ok := m.list.SelectedItem().(themeItem); ok {
        if selected.name != appstyles.Active.Name {
            appstyles.SetTheme(selected.name)
            // Re-render the delegate with the new active theme's
            // name so the "● current" marker moves.
            d := m.list.Delegate().(themePickerDelegate)
            d.currentTheme = appstyles.Active.Name
            m.list.SetDelegate(d)
        }
    }

    return m, listCmd
}

func (m ThemePickerModalModel) View() tea.View {
    hints := renderKeyHints([]KeyHint{
        hintFor(keys.List.Navigate),
        hintAs(keys.Overlay.Submit, "close"),
        hintAs(keys.Overlay.Cancel, "close"),
    }, appstyles.Active.TextMuted)

    content := lipgloss.JoinVertical(lipgloss.Left,
        m.list.View(),
        "",
        hints,
    )

    return tea.NewView(modalSurface(appstyles.Active.ModalBg, content))
}

// ThemePickerModal builds the picker. The cursor starts on whichever theme
// is currently active, so the user sees their starting point before moving.
func ThemePickerModal() tea.Model {
    // Sorted theme names for stable list order.
    names := sortedThemeNames()

    items := make([]list.Item, 0, len(names))
    for _, name := range names {
        items = append(items, themeItem{name: name})
    }

    cl := list.New(
        items,
        themePickerDelegate{currentTheme: appstyles.Active.Name},
        40,
        len(items)+2,
    )
    cl.SetShowHelp(false)
    cl.SetShowStatusBar(false)
    cl.SetShowPagination(false)
    cl.Title = "Theme"
    cl.Styles.Title = cl.Styles.Title.Background(appstyles.Active.Accent)

    // Land the cursor on the active theme.
    for i, name := range names {
        if name == appstyles.Active.Name {
            cl.Select(i)
            break
        }
    }

    return ThemePickerModalModel{
        list:         cl,
        originalName: appstyles.Active.Name,
    }
}

// sortedThemeNames returns every registered theme name in sorted order,
// for stable list presentation across runs.
func sortedThemeNames() []string {
    // import "slices", "maps" at the top of the file
    return slices.Sorted(maps.Keys(appstyles.Themes))
}
```

**Note on the live-preview pattern.** The modal assigns `appstyles.Active`
directly on cursor move rather than emitting a command that AppModel handles.
This is deliberate: the theme change must be visible on the very next render
frame, and going through AppModel would add one frame of latency. Since
`appstyles.Active` is a package-level variable (not protected state), a direct
assignment is safe and consistent with the design doc's own description:
"assign a different registered Theme to Active and the next frame draws it."

### 6. `src/appstyles/Theme.go` — export `SetTheme` for the modal

The modal calls `appstyles.SetTheme` (exported). This is the same function
described in section 1 — the modal uses the exported name, internal code
uses the unexported one if needed.

```go
// SetTheme assigns a new active theme. Exported so the theme picker modal
// can apply themes live as the cursor moves. Returns nil today; when
// config file persistence lands, it returns a command that writes the
// name to disk.
func SetTheme(name string) tea.Cmd {
    if t, ok := Themes[name]; ok {
        Active = t
    }
    return nil
}
```

One function, two callers today (the modal on cursor-move, the modal on
close). When persistence lands, the return type gains a command.

### 7. `src/model/Update.go` — handle `OpenThemePickerModalMsg`

Add one case, following the exact pattern of `OpenAboutModalMsg`:

```go
case cmds.OpenThemePickerModalMsg:
    m.activeModal = components.ThemePickerModal()
```

And handle `ThemeChangedMsg`:

```go
case cmds.ThemeChangedMsg:
    // The theme was already applied live by the modal. When config file
    // persistence lands, this handler writes msg.Name to disk.
    // For now, nothing to do.
```

### 8. `src/model/Update.go` — handle `T` key

In the `tea.KeyPressMsg` handler, after the existing global key matches
(`keys.Global.Quit`, `keys.Global.Help`, `keys.Global.About`), add:

```go
case key.Matches(msg, keys.Global.Theme):
    finalCmds = append(finalCmds, cmds.OpenThemePickerModal())
```

This sits inside the `keyboardOwned()` guard (same as About), so `T` is a
letter while a filter is being typed, not a command.

### 9. Tests

#### 9a. `src/components/ThemePickerModal_test.go` (new file)

```go
package components

import (
    "testing"

    tea "charm.land/bubbletea/v2"
    "github.com/filipemolina/cais/src/appstyles"
    "github.com/filipemolina/cais/src/cmds"
)

func TestThemePickerOpensOnActiveTheme(t *testing.T) {
    original := appstyles.Active
    defer func() { appstyles.Active = original }()

    appstyles.Active = appstyles.Themes["stitcher-light"]

    m := ThemePickerModal().(ThemePickerModalModel)

    selected, ok := m.list.SelectedItem().(themeItem)
    if !ok {
        t.Fatal("no item selected on open")
    }
    if selected.name != "stitcher-light" {
        t.Errorf("cursor on %q, want stitcher-light", selected.name)
    }
}

func TestThemePickerLivePreviewOnCursorMove(t *testing.T) {
    original := appstyles.Active
    defer func() { appstyles.Active = original }()

    appstyles.Active = appstyles.Themes["stitcher-dark"]

    m := ThemePickerModal().(ThemePickerModalModel)

    // Move cursor down — should land on the other theme and apply it.
    updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown, Text: "down"})
    m = updated.(ThemePickerModalModel)

    // The active theme should have changed.
    if appstyles.Active.Name == "stitcher-dark" {
        t.Error("live preview did not change the active theme after cursor move")
    }
}

func TestThemePickerEscClosesWithThemeChanged(t *testing.T) {
    original := appstyles.Active
    defer func() { appstyles.Active = original }()

    m := ThemePickerModal().(ThemePickerModalModel)

    _, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc, Text: "esc"})

    msgs := collectMessages(t, cmd)

    // Should contain a CloseModalMsg (to dismiss the picker).
    if !hasMessageOfType[cmds.CloseModalMsg](t, cmd) {
        t.Fatalf("expected CloseModalMsg on Esc, got %v", msgs)
    }
}

func TestThemePickerEnterClosesWithThemeChanged(t *testing.T) {
    original := appstyles.Active
    defer func() { appstyles.Active = original }()

    m := ThemePickerModal().(ThemePickerModalModel)

    _, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "enter"})

    if !hasMessageOfType[cmds.CloseModalMsg](t, cmd) {
        t.Fatal("expected CloseModalMsg on Enter")
    }
}
```

#### 9b. `src/keys/Keys_test.go` — add theme key to existing pin tests

The existing `TestFooterHints` pins every context. No change needed —
`T` is not in `Active()` or `Globals()`, so it does not appear in any
footer context.

Add a test that `T` appears in `Catalog()`'s Global scope:

```go
func TestCatalogContainsThemeKey(t *testing.T) {
    ctx := Context{Page: "Home", Focused: constants.COMPONENT_BODY_LIST}
    scopes := Catalog(ctx)

    found := false
    for _, scope := range scopes {
        if scope.Title != "Global" {
            continue
        }
        for _, entry := range scope.Entries {
            if entry.Binding == Global.Theme {
                found = true
                if !entry.Available {
                    t.Error("Theme should always be available in Global scope")
                }
            }
        }
    }
    if !found {
        t.Error("Theme key not found in Catalog Global scope")
    }
}
```

#### 9c. `src/model/background_test.go` — add theme picker modal case

Add to `TestNoBackgroundBleedInModals`:

```go
{
    name: "theme picker modal",
    msgs: []tea.Msg{
        cmds.GetConfigMsg{FileName: "compose.yaml", Project: project()},
        cmds.OpenThemePickerModalMsg{},
    },
},
```

#### 9d. E2E rig test (optional)

```go
func TestRigThemePicker(t *testing.T) {
    setupProjectDir(t)

    r := newRig(t)
    if !r.WaitFor("core", 3*time.Second) {
        t.Fatal("groups never rendered")
    }

    r.Send(letterKey('T'))

    if !r.WaitFor("Theme", 3*time.Second) {
        t.Fatalf("theme picker never opened. Output:\n%s", r.Output())
    }

    // Press Esc — modal closes.
    r.Send(keyPress(tea.KeyEsc))

    if !r.WaitForNot("Theme", 3*time.Second) {
        t.Fatal("theme picker did not close on Esc")
    }
}
```

### 10. Documentation updates

#### 10a. `README.md`

Add `T` to the keybindings table under the Global scope:

```
| `T` | Open the theme picker | Everywhere except while typing |
```

Update the keybindings paragraph that mentions `?` overlay to include `T`.

#### 10b. `docs/DESIGN.md`

Update the "Where keybindings live" section:
- Add `T` to the Global tier in the tiers table.
- Mention `T` in the paragraph about `a` (About): "Similarly, `T` opens
  the theme picker — advertised in the help overlay, not the footer."

Update the "Color lives on a Theme" section:
- Note that `appstyles.SetTheme` is the exported seam the picker uses.
- Note that persistence is deferred to the config file work.

#### 10c. `TODO.md`

Add a completed entry under "Suggested next steps":

```markdown
- [x] **[S] Theme picker modal** — `T` opens a list of registered
  themes; cursor movement applies the theme live (assigns
  `appstyles.Active`, the next frame repaints). Esc or Enter closes
  the modal, keeping the active theme. Persistence is deferred to the
  config file work — the seam is `appstyles.SetTheme`, which gains a
  disk write when `~/.config/cais/config.yaml` lands.
```

#### 10d. `docs/ROADMAP.md`

Remove the theme picker from the "Explicitly post-alpha" section. It's done.
Keep "additional themes" (catpuccin, nord, etc.) as post-alpha — that's
adding entries to `appstyles.Themes`, which the picker then picks up
automatically.

## Implementation order

Each step compiles and passes tests on its own.

| Step | File | What | Risk |
|------|------|------|------|
| 1 | `src/appstyles/Theme.go` | Add `SetTheme()` exported function | Trivial, no callers yet |
| 2 | `src/keys/Keys.go` | Add `Theme` binding to `GlobalKeys`, `Global`, `pressableNow`, `Catalog` | New binding, no handlers yet |
| 3 | `src/cmds/OpenThemePickerModal.go` | New message type + command | New file |
| 4 | `src/cmds/ThemeChangedMsg.go` | New message type + command | New file |
| 5 | `src/components/ThemePickerModal.go` | New component | New file, ~120 lines |
| 6 | `src/model/Update.go` | Wire up `T` key + both modal messages | ~5 lines added |
| 7 | Tests | Component, keys, background-bleed, optional rig | New + additions |
| 8 | Docs | README, DESIGN.md, TODO.md, ROADMAP.md | Prose only |

## Acceptance criteria

1. Press `T` from any page (not while typing) → theme picker modal opens.
2. The cursor starts on the currently active theme.
3. Move cursor down/up → the entire TUI repaints in the new theme live.
4. Press `Esc` or `Enter` → modal closes, active theme is whatever was
   last previewed.
5. `T` is listed in the `?` help overlay under the Global scope.
6. `T` does NOT appear in the footer (width-constrained).
7. While typing a filter, `T` is a letter, not a command.
8. `go build ./... && go vet ./... && go test ./... && gofmt -l .` all green.
9. `go test -race ./...` green.
10. Background-bleed tests pass under both themes with the modal open.

## Adding more themes later

To add a new theme (e.g., `catppuccin-mocha`), add one entry to
`appstyles.Themes` in `src/appstyles/Theme.go`:

```go
"catppuccin-mocha": newTheme(themeParams{
    Name:   "catppuccin-mocha",
    Dark:   true,
    Accent: lipgloss.Color("#CBA6F7"),
    Text:   lipgloss.Color("#CDD6F4"),
    Panel:  lipgloss.Color("#1E1E2E"),
    Modal:  lipgloss.Color("#313244"),
    Danger: lipgloss.Color("#F38BA8"),

    Running:  lipgloss.Color("#A6E3A1"),
    Stopped:  lipgloss.Color("#6C7086"),
    Starting: lipgloss.Color("#F9E2AF"),
    Err:      lipgloss.Color("#F38BA8"),
}),
```

The picker picks it up automatically — it iterates `appstyles.Themes`, so
no other file changes.
