# Plan: UX Improvements

Four changes to improve the user experience. This document covers scope,
implementation order, affected files, and testing strategy.

## Changes

1. **Auto-select on navigation** — Arrow keys select the item under the cursor
2. **`n` works on both panels** — Create group from list or details panel
3. **Action feedback with spinner** — Visual feedback while docker actions run
4. **Error modals for foreground errors** — Modal dialog instead of banner for action errors

## Implementation Order

The changes are ordered by dependency and risk:

| # | Change | Effort | Lines | Risk | Depends on |
|---|--------|--------|-------|------|------------|
| 1 | Auto-select on navigation | Low | ~50 | Low | — |
| 2 | `n` on both panels | Low | ~20 | Low | — |
| 3 | Action feedback (spinner) | Medium | ~90 | Low | #1 |
| 4 | Error modals (foreground) | Medium | ~100 | Medium | — |

**Total: ~260 lines changed**

Changes #1 and #2 are independent and can be done in parallel. Change #3
depends on #1 (the panel needs to know what's selected to show the right
spinner). Change #4 is independent but benefits from #3 being done first
(the spinner handles the "loading" state, the modal handles the "failed"
state).

---

## Change 1: Auto-select on navigation

### Goal

Arrow keys in the list automatically select the item under the cursor,
updating the details panel immediately. No separate "space to select" step.

### Current flow

```
Arrow keys → cursor moves → Space → SetSelectedGroup/SetSelectedService → details update
```

### New flow

```
Arrow keys → cursor moves → SetSelectedGroup/SetSelectedService → details update
```

### Files to modify

| File | Change |
|------|--------|
| `src/components/GroupsList.go` | Detect cursor change, emit `SetSelectedGroup` |
| `src/components/ServicesList.go` | Detect cursor change, emit `SetSelectedService` |
| `src/components/GroupDetailsPanel.go` | Update empty-state hint text |
| `src/components/DetailsPanel.go` | Update empty-state hint text |
| `src/keys/Keys.go` | Keep `List.Select` as alias for start action |
| `src/model/selection_test.go` | Update tests for auto-select behavior |

### Implementation details

#### GroupsList.go

In `Update`, after `m.list.Update(msg)`, detect cursor change and emit selection:

```go
// Track cursor before the list processes the key
previousIndex := m.list.Index()

// Let the inner list process the arrow key
if m.isFocused {
    var cmd tea.Cmd
    m.list, cmd = m.list.Update(msg)
    finalCmds = append(finalCmds, cmd)
}

// If cursor moved, auto-select the item
if m.list.Index() != previousIndex {
    if item := m.list.SelectedItem(); item != nil {
        if group, ok := item.(apptypes.GroupListItem); ok {
            m.activeGroup = string(group)
            m.syncActiveIndex()
            finalCmds = append(finalCmds, cmds.SetSelectedGroup(string(group)))
        }
    }
}
```

#### ServicesList.go

Same pattern as GroupsList, but for services:

```go
previousIndex := m.list.Index()

if m.isFocused {
    var cmd tea.Cmd
    m.list, cmd = m.list.Update(msg)
    finalCmds = append(finalCmds, cmd)
}

if m.list.Index() != previousIndex {
    if item := m.list.SelectedItem(); item != nil {
        if service, ok := item.(apptypes.ServiceListItem); ok {
            m.activeService = service.Service.Name
            m.syncActiveIndex()
            finalCmds = append(finalCmds, cmds.SetSelectedService(service.Service))
        }
    }
}
```

#### Hint text updates

In `GroupDetailsPanel.renderBody`, change the empty-state hint:

```go
// Before
"↑/↓", "then space"

// After
"↑/↓", "to browse"
```

Same in `DetailsPanel.View` for the "Select a service" empty state.

#### Space/Enter behavior

Keep `List.Select` but repurpose it as "start the selected item":

```go
case key.Matches(msg, keys.List.Select):
    if m.activeGroup != "" {
        finalCmds = append(finalCmds, cmds.RequestDockerAction("start", m.activeGroup, true))
    }
```

This gives users a familiar action key while keeping the auto-select flow.

### Testing

- Add test: cursor movement triggers selection message
- Add test: space/enter starts the selected item
- Update `TestReloadKeepsTheSelectedService` (selection now happens on cursor move)

---

## Change 2: `n` works on both panels

### Goal

The `n` key opens the create group modal from either the list or details
panel on the Home page. This fixes the onboarding issue where the empty
state says "press n" but the key only works on one panel.

### Files to modify

| File | Change |
|------|--------|
| `src/model/Update.go` | Handle `n` at app level for Home page |
| `src/components/GroupsList.go` | Remove `n` handler (now in AppModel) |
| `src/keys/Keys.go` | Move `List.New` to page-scoped context |
| `src/model/keyboard_test.go` | Add test for `n` from details panel |

### Implementation details

#### AppModel.Update

In the `tea.KeyPressMsg` case, after the modal/keyboardOwned checks and
before the panel-specific key handling:

```go
// n creates a group from either panel on Home.
case key.Matches(msg, keys.List.New):
    if m.activePage == "Home" {
        finalCmds = append(finalCmds, cmds.OpenCreateGroupModal())
    }
```

This works because:
- The modal check above prevents it from firing while a modal is open
- The `keyboardOwned()` guard prevents it from firing while filtering
- The page check ensures it only works on Home

#### GroupsList.go

Remove the `n` handler from the `tea.KeyPressMsg` case:

```go
// Remove this block
case key.Matches(msg, keys.List.New):
    finalCmds = append(finalCmds, cmds.OpenCreateGroupModal())
```

#### keys/Keys.go

The `List.New` binding stays in the `List` struct but is now handled at
the app level. The `Active` function should still return it for the
footer when on Home, regardless of which panel is focused.

Update `Active` to include `List.New` in both panel contexts for Home:

```go
case "Home":
    switch ctx.Focused {
    case constants.COMPONENT_BODY_LIST:
        own := []key.Binding{List.New}
        // ... existing code

    case constants.COMPONENT_BODY_DETAILS:
        if !ctx.Selected {
            return []key.Binding{List.New, Global.Back, Global.NextPanel}
        }
        return []key.Binding{
            List.New,
            Details.Start, Details.Stop, Details.Restart,
            Details.Pull, Details.Remove, Details.Logs,
            Global.Back, Global.NextPanel,
        }
    }
```

### Testing

- Add test: `n` opens create modal when details panel is focused
- Add test: `n` does nothing on Services page
- Add test: `n` does nothing while a modal is open

---

## Change 3: Action feedback with spinner

### Goal

Show a spinning animation while docker actions (start, stop, restart, pull,
remove) are in progress. The spinner appears in two locations:
1. Title pill area (replaces status pill)
2. Action buttons area (replaces buttons)

### Files to modify

| File | Change |
|------|--------|
| `src/components/GroupDetailsPanel.go` | Add spinner state, render spinner in title and buttons |
| `src/components/DetailsPanel.go` | Same spinner for services |
| `src/model/AppModel.go` | Add `pendingAction` field |
| `src/model/Update.go` | Handle `SetPendingActionMsg`, disable keys while pending |
| `src/keys/Keys.go` | Add `pendingAction` to Context, disable action keys |
| `src/cmds/SetPendingAction.go` | New command file |
| `src/components/spinner.go` | New spinner helper |
| `src/model/action_feedback_test.go` | New test file |

### Implementation details

#### New command: SetPendingAction

```go
// src/cmds/SetPendingAction.go
package cmds

import tea "charm.land/bubbletea/v2"

type SetPendingActionMsg struct {
    Action string
    Target string
    IsGroup bool
}

func SetPendingAction(action string, target string, isGroup bool) tea.Cmd {
    return func() tea.Msg {
        return SetPendingActionMsg{Action: action, Target: target, IsGroup: isGroup}
    }
}

type ClearPendingActionMsg struct{}

func ClearPendingAction() tea.Cmd {
    return func() tea.Msg { return ClearPendingActionMsg{} }
}
```

#### AppModel changes

Add field to track pending action:

```go
type AppModel struct {
    // ... existing fields
    pendingAction *PendingAction
}

type PendingAction struct {
    Action  string
    Target  string
    IsGroup bool
}
```

Update `RunDockerActionMsg` handler:

```go
case cmds.RunDockerActionMsg:
    // Set pending state and show spinner
    m.pendingAction = &PendingAction{
        Action:  msg.Action,
        Target:  msg.Target,
        IsGroup: msg.IsGroup,
    }
    finalCmds = append(finalCmds, cmds.SetPendingAction(msg.Action, msg.Target, msg.IsGroup))
    finalCmds = append(finalCmds, cmds.RunDockerAction(
        msg.Action, msg.Target, msg.IsGroup, m.config.configFileName,
    ))
```

Update `DockerActionMsg` handler:

```go
case cmds.DockerActionMsg:
    // Clear pending state
    m.pendingAction = nil
    finalCmds = append(finalCmds, cmds.ClearPendingAction())
    // ... existing error handling
```

#### Spinner component

Create a shared spinner helper:

```go
// src/components/spinner.go
package components

import (
    "charm.land/bubbles/v2/spinner"
    "github.com/filipemolina/cais/src/appstyles"
)

func newSpinner() spinner.Model {
    s := spinner.New()
    s.Spinner = spinner.Points
    s.Style = appstyles.Active.Accent
    return s
}

// ActionDescription returns a human-readable description of the action.
func ActionDescription(action, target string, isGroup bool) string {
    kind := "service"
    if isGroup {
        kind = "group"
    }

    switch action {
    case "start":
        return "Starting %s %q..."
    case "stop":
        return "Stopping %s %q..."
    case "restart":
        return "Restarting %s %q..."
    case "pull":
        return "Pulling image for %s %q..."
    case "remove":
        return "Removing %s %q..."
    default:
        return "Running %s on %s %q..."
    }
    // (formatted with kind, target)
}
```

#### GroupDetailsPanel changes

Add fields for spinner state:

```go
type GroupDetailsPanelModel struct {
    // ... existing fields
    pendingAction *PendingAction
    spinner       spinner.Model
}
```

Add message handling:

```go
case cmds.SetPendingActionMsg:
    m.pendingAction = &PendingAction{Action: msg.Action, Target: msg.Target, IsGroup: msg.IsGroup}
    m.spinner, cmd = m.spinner.Tick()
    finalCmds = append(finalCmds, cmd)

case cmds.ClearPendingActionMsg:
    m.pendingAction = nil

case spinner.TickMsg:
    if m.pendingAction != nil {
        m.spinner, cmd = m.spinner.Update(msg)
        finalCmds = append(finalCmds, cmd)
    }
```

Update `titlePill` to show spinner:

```go
func (m GroupDetailsPanelModel) titlePill() string {
    if m.pendingAction != nil {
        return m.spinner.View() + " " + actionLabel(m.pendingAction.Action) + "..."
    }

    // ... existing status pill logic
}
```

Update `renderBody` to show spinner in buttons area:

```go
func (m GroupDetailsPanelModel) renderBody() string {
    // ... existing header and table rendering

    if m.pendingAction != nil {
        buttons = m.renderPendingAction(bodyWidth, bg)
    } else {
        buttons = renderActionButtons(bodyWidth, bg)
    }

    // ... rest of rendering
}

func (m GroupDetailsPanelModel) renderPendingAction(width int, bg color.Color) string {
    desc := fmt.Sprintf("%s %s %q...",
        actionLabel(m.pendingAction.Action),
        kindLabel(m.pendingAction.IsGroup),
        m.pendingAction.Target,
    )

    style := lipgloss.NewStyle().
        Foreground(appstyles.Active.TextPrimary).
        Background(bg).
        Width(width).
        AlignHorizontal(lipgloss.Center)

    return style.Render(m.spinner.View() + " " + desc)
}
```

#### Disable action keys while pending

Update `keys.Active` to check for pending action:

```go
type Context struct {
    // ... existing fields
    PendingAction bool
}

// In Active(), when PendingAction is true, return only Back and NextPanel:
case constants.COMPONENT_BODY_DETAILS:
    if ctx.PendingAction {
        return []key.Binding{Global.Back, Global.NextPanel}
    }
    // ... existing action keys
```

Update the footer to not show action keys while pending. The `?` overlay
should show them as unavailable (dimmed).

### Testing

- Add test: `SetPendingActionMsg` sets pending state
- Add test: `ClearPendingActionMsg` clears pending state
- Add test: action keys are disabled while pending
- Add test: spinner renders in title pill area
- Add test: spinner renders in buttons area

---

## Change 4: Error modals for foreground errors

### Goal

Show foreground errors (from docker actions, config loads, etc.) in a modal
dialog instead of the banner. Background poll errors keep the banner to
avoid modal fatigue.

### Files to modify

| File | Change |
|------|--------|
| `src/components/ErrorModal.go` | New modal component |
| `src/cmds/OpenErrorModal.go` | New command file |
| `src/model/AppModel.go` | Change error handling for foreground errors |
| `src/model/Update.go` | Open modal instead of setting `lastError` for foreground |
| `src/model/View.go` | Remove `errorBannerStyle` (keep for background only) |
| `src/model/error_banner_test.go` | Update tests |

### Implementation details

#### New component: ErrorModal

```go
// src/components/ErrorModal.go
package components

import (
    "github.com/filipemolina/cais/src/appstyles"
    "github.com/filipemolina/cais/src/cmds"
    "github.com/filipemolina/cais/src/keys"

    "charm.land/bubbles/v2/key"
    tea "charm.land/bubbletea/v2"
)

type ErrorModalModel struct {
    message string
}

func (m ErrorModalModel) Init() tea.Cmd {
    return nil
}

func (m ErrorModalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    keyMsg, ok := msg.(tea.KeyPressMsg)
    if !ok {
        return m, nil
    }

    switch {
    case key.Matches(keyMsg, keys.Overlay.Yes),
         key.Matches(keyMsg, keys.Overlay.Cancel):
        return m, cmds.CloseModal(nil)
    }

    return m, nil
}

func (m ErrorModalModel) View() tea.View {
    content := "Error\n\n" + m.message + "\n\nPress esc to dismiss"
    return tea.NewView(modalSurface(appstyles.Active.ModalBg, content))
}

func ErrorModal(message string) tea.Model {
    return ErrorModalModel{message: message}
}
```

#### New command: OpenErrorModal

```go
// src/cmds/OpenErrorModal.go
package cmds

import tea "charm.land/bubbletea/v2"

type OpenErrorModalMsg struct {
    Message string
}

func OpenErrorModal(message string) tea.Cmd {
    return func() tea.Msg {
        return OpenErrorModalMsg{Message: message}
    }
}
```

#### AppModel.Update changes

Distinguish between foreground and background errors:

```go
case cmds.GetRunningContainersMsg:
    if msg.Err != nil {
        if msg.Background {
            // Background errors: keep the banner (less disruptive)
            m.lastError = msg.Err.Error()
            m.lastErrorFromPoll = true
        } else {
            // Foreground errors: show a modal
            if m.activeModal == nil {
                m.activeModal = components.ErrorModal(msg.Err.Error())
            }
        }
    } else {
        // Success: clear both banner and any pending error state
        if !msg.Background || m.lastErrorFromPoll {
            m.lastError = ""
            m.lastErrorFromPoll = false
        }
    }
```

Same pattern for `DockerActionMsg`:

```go
case cmds.DockerActionMsg:
    m.pendingAction = nil
    finalCmds = append(finalCmds, cmds.ClearPendingAction())

    if msg.Err != nil {
        // Docker action errors are always foreground
        if m.activeModal == nil {
            m.activeModal = components.ErrorModal(msg.Err.Error())
        }
    } else {
        m.lastError = ""
        finalCmds = append(finalCmds, cmds.GetRunningContainers(m.config.configFileName))
    }
```

And for other foreground errors:

```go
case cmds.EditGroupMsg:
    if msg.Err != nil {
        if m.activeModal == nil {
            m.activeModal = components.ErrorModal(msg.Err.Error())
        }
    } else {
        m.lastError = ""
        finalCmds = append(finalCmds, cmds.GetConfig(m.config.source))
    }
```

#### View.go changes

The `errorBannerStyle` and banner rendering stay for background errors.
No changes needed here — the banner is still used for background errors.

#### Remove Esc-to-dismiss-banner for foreground errors

The existing Esc ladder (banner → back to list) only applies to background
errors now. Foreground errors are dismissed by the modal's own Esc handler.

Update the Esc handler in `Update`:

```go
case key.Matches(msg, keys.Global.Back):
    // Only dismiss background error banners with Esc
    if m.lastError != "" && m.lastErrorFromPoll && !m.escKept() {
        m.lastError = ""
        m.lastErrorFromPoll = false
        break
    }
    // ... existing back-to-list logic
```

### Testing

- Update `TestEscDismissesAForegroundErrorBanner` — foreground errors now use modals
- Add test: foreground docker error opens modal
- Add test: background poll error keeps banner
- Add test: modal blocks key input while open
- Add test: Esc in modal dismisses it

---

## Testing Strategy

### Unit tests

Each change has its own test file or updates to existing tests:

| Change | Test file | Key tests |
|--------|-----------|-----------|
| Auto-select | `selection_test.go` | Cursor movement triggers selection |
| `n` on both panels | `keyboard_test.go` | `n` works from details panel |
| Spinner | `action_feedback_test.go` | Pending state transitions, spinner renders |
| Error modals | `error_banner_test.go` | Modal opens for foreground, banner for background |

### E2E tests (rig)

Extend `rig_test.go` with end-to-end flows:

1. **Auto-select flow**: Start app → arrow down → verify details panel updates
2. **Create from details**: Focus details → press `n` → verify modal opens
3. **Action feedback**: Press `s` → verify spinner appears → wait for completion → verify spinner disappears
4. **Error modal**: Trigger a docker error → verify modal appears → press esc → verify modal closes

### Manual testing checklist

- [ ] Arrow keys auto-select items in both lists
- [ ] `n` works from both panels on Home
- [ ] `n` does nothing on Services or Files pages
- [ ] Spinner appears immediately after pressing action key
- [ ] Spinner animates smoothly (no jitter)
- [ ] Spinner disappears when action completes
- [ ] Action keys are disabled while spinner is active
- [ ] Footer updates to reflect disabled keys
- [ ] Foreground errors show in modal
- [ ] Background errors show in banner
- [ ] Modal blocks key input
- [ ] Esc dismisses error modal
- [ ] Esc dismisses background error banner

---

## Future enhancements (out of scope)

- **Toast notifications**: Replace background error banner with auto-dismissing toast
- **Action queue**: Allow queuing actions instead of ignoring while pending
- **Action cancellation**: Allow cancelling an in-progress action
- **Progress indication**: Show actual progress (e.g., "Pulling 2/5 layers...")
