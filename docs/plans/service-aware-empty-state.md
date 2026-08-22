# Service-Aware Empty State — Implementation Plan

> **SUPERSEDED** (2026-08-22): the service overview this plan describes was
> removed when the reserved `ungrouped` group landed
> (`docs/plans/ungrouped-services.md`, P2). A file with services always has at
> least the `ungrouped` row to select, and selecting it shows the same member
> table with a real header card and working action keys — the overview was the
> worse version of the screen it was standing in for. This document is kept
> for historical reference only.

> **Before you start.** Work on a feature branch of small commits, merged
> `--no-ff`; `go build ./... && go vet ./... && go test ./... && gofmt -l .`
> green at **every** commit, not just at the tip — `docs/ROADMAP.md`
> §Conventions is the full contract and `CONTRIBUTING.md` explains how a TUI
> gets tested. Behaviour that only shows on screen gets checked in the real app
> with VHS before it is committed. **Step 3 of the post-alpha order** — after `image-search.md` Phase 1, because the empty state's copy depends on `n` being able to add a service.

## Problem

When a user opens Cais for the first time with a compose file that
has services but no `profiles:` tags, the Groups page shows:

- **Left panel:** "No groups yet. Press `n` to create one, or add profiles to
  services in your compose file."
- **Right panel:** A "Getting started" card explaining what groups are.

The user has no way to see *what services exist* without navigating to the
Services page (`2`). This is the discoverability gap the original proposal
aimed at: "the first time the user opens the TUI, if there are services
running but no groups, the user can use the side pane to select the services,
then use the `n` shortcut to create the group."

## Solution

Replace the static "Getting started" card in the **right (details) panel**
with a live summary of the compose file's services when no groups exist. The
card shows the service count, and each service's name, image, and running
state — read-only, no action keys, no multi-select. The `n` key on the left
panel still opens the existing two-step modal flow (name → service checklist),
which already shows every service with checkboxes.

```
┌ Details ──────────────────────────────────────────────────────┐
│                                                               │
│   12 services in compose.yml — no groups yet                  │
│   ────────────────────────────────────────────────            │
│    ● radarr          linuxserver/radarr:latest    running     │
│    ● sonarr          linuxserver/sonarr:latest    running     │
│    ● plex            plexinc/pms-docker:latest    stopped     │
│    ○ traefik         traefik:v3                   running     │
│    ○ portainer       portainer/portainer-ce       stopped     │
│    …                                                          │
│                                                               │
│   Press n to create your first group, or 2 to browse.        │
│                                                               │
└───────────────────────────────────────────────────────────────┘
```

- `●` = running (green), `○` = stopped (dim). Same dot colors used in the
  group member table (`GroupDetailsPanel.renderMemberRow`).
- Services sorted alphabetically, same order as `configSyncCmds`.
- List truncated to fit the panel height with `…` at the bottom when there
  are more services than rows.
- The footer hint at the bottom points the user at `n` (create group) and `2`
  (navigate to Services page for full details).

### What does NOT change

| Aspect | Status |
|--------|--------|
| Groups list (left panel) | **Unchanged.** Shows only groups, empty state text stays as-is. |
| `n` key flow | **Unchanged.** Opens `GroupNameModal` → `ServiceChecklistModal`, which already shows all services. |
| `e` / `d` keys | **Unchanged.** Require a group to be focused. |
| Services page | **Unchanged.** Retains its purpose (full details, inline YAML editor, per-service actions). |
| Keymap (`src/keys`) | **Unchanged.** No new bindings, no context changes. |
| Footer (`KeybindingBar`) | **Unchanged.** The `keys.Context` struct and `Active()` function see no new signals. |
| Help overlay | **Unchanged.** |
| `helpContext()` | **Unchanged.** `ListEmpty` still means "no groups." |
| `SetSelectedGroupMsg` / selection model | **Unchanged.** No group is selected when the empty state shows. |
| Docker actions | **Unchanged.** No action keys work from the empty state — it's read-only. |

## Scope and risk

This is a **low-risk, additive change to one component**. No new messages, no
new state fields on `AppModel`, no keybinding changes, no data flow changes.
The data the panel needs (`services`, `containers`) already reaches it via
existing messages — it just doesn't use them in the empty state today.

### Why this is not a regression

- The existing onboarding text ("Groups are Compose profiles…") is replaced,
  but the same information is still reachable: pressing `n` opens the name
  modal which explains the concept implicitly, and the checklist in step 2
  shows every service.
- The "Getting started" card is a static string. The new card is dynamic and
  strictly more informative.
- No existing test asserts the exact text of the "Getting started" card (the
  background-bleed tests check pixel coverage, not prose).

## Detailed changes

### 1. `src/components/GroupDetailsPanel.go`

This is where almost all the work happens.

#### 1a. Data fields already present — no additions needed

The model already stores everything required:

```go
type GroupDetailsPanelModel struct {
    selectedGroup string
    services      []types.ServiceConfig      // ← all services (set by SetServicesListMsg)
    containers    []apptypes.DockerContainer // ← live state (set by GetRunningContainersMsg)
    panelWidth    int
    panelHeight   int
    isFocused     bool
    componentId   int
}
```

Both `services` and `containers` are populated by `configSyncCmds` + the
foreground `GetRunningContainers` that runs on config load. They arrive via
`cmds.SetServicesListMsg` and `cmds.GetRunningContainersMsg`, which the panel
already handles.

#### 1b. New helper: `standaloneServices()`

Returns services with zero profile tags, sorted alphabetically (they arrive
pre-sorted from `configSyncCmds`, but the sort is defensive):

```go
// standaloneServices returns services that carry no profiles: tag. These
// are the services the user has not yet grouped. Used only by the empty
// state card when no groups exist — once groups exist, these services
// are reachable through the Services page.
func (m GroupDetailsPanelModel) standaloneServices() []types.ServiceConfig {
    var out []types.ServiceConfig
    for _, svc := range m.services {
        if len(svc.Profiles) == 0 {
            out = append(out, svc)
        }
    }
    return out
}
```

#### 1c. New helper: `renderServiceOverviewCard()`

Replaces the call to `renderEmptyCard` in the "no groups" branch of
`renderBody`. It is a new method on `GroupDetailsPanelModel`, not a new
variant of `renderEmptyCard`, because it renders a scrollable list rather
than a centered card.

```go
// renderServiceOverviewCard replaces the "Getting started" card when the
// compose file has services but no groups. It lists every service with its
// running state so the user knows what's in their stack before they start
// grouping, and points them at n (create) and 2 (Services page).
func (m GroupDetailsPanelModel) renderServiceOverviewCard(width, availHeight int, bg color.Color) string {
    cardBg := appstyles.Active.BackgroundRecessed

    // Header: count + separator
    total := len(m.services)
    standalone := m.standaloneServices()
    headerText := fmt.Sprintf("%d %s in compose.yml — no groups yet",
        total, pluralN(total, "service"))
    headerStyle := lipgloss.NewStyle().
        Bold(true).
        Foreground(appstyles.Active.TextPrimary).
        Background(cardBg).
        Width(width)
    rule := lipgloss.NewStyle().
        Foreground(appstyles.Active.BorderDefault).
        Background(cardBg).
        Width(width).
        Render(strings.Repeat("─", max(width, 0)))

    // Service rows: one per service, truncated to fit.
    // Reserve rows for: header (1) + blank (1) + rule (1) + blank (1) +
    // footer hint (1) + blank (1) = 6 rows of overhead.
    rowBudget := availHeight - 6
    if rowBudget < 1 {
        rowBudget = 1
    }

    rows := m.renderServiceRows(standalone, width, rowBudget)

    // Footer hint
    hintStyle := lipgloss.NewStyle().
        Foreground(appstyles.Active.TextDim).
        Background(cardBg).
        Width(width)
    keyStyle := lipgloss.NewStyle().
        Bold(true).
        Foreground(appstyles.Active.Accent).
        Background(cardBg)
    footer := keyStyle.Render("n") +
        hintStyle.Render(" create group · ") +
        keyStyle.Render("2") +
        hintStyle.Render(" browse services")

    content := lipgloss.JoinVertical(lipgloss.Left,
        headerStyle.Render(headerText),
        "",
        rule,
        "",
        rows,
        "",
        footer,
    )

    card := lipgloss.NewStyle().
        Width(width).
        Background(cardBg).
        Padding(1, 2).
        Border(lipgloss.RoundedBorder()).
        BorderForeground(appstyles.Active.BorderCard).
        BorderBackground(cardBg).
        Render(content)

    card = appstyles.FillBackground(cardBg, card)

    return lipgloss.NewStyle().
        Width(width).
        Height(availHeight).
        MaxHeight(availHeight).
        Background(bg).
        AlignHorizontal(lipgloss.Center).
        AlignVertical(lipgloss.Center).
        Render(card)
}
```

#### 1d. New helper: `renderServiceRows()`

Renders the per-service rows, reusing the same dot/NAME/IMAGE/STATE columns
as the member table (but without HEALTH/UPTIME/PORTS — those are detail
concerns for the Services page). Truncates with `…` when services exceed
the row budget.

```go
// renderServiceRows renders one row per standalone service, using the
// member-table dot/name/image/state columns. rowBudget is the max number
// of rows; when services exceed it, the last row is an ellipsis indicator.
func (m GroupDetailsPanelModel) renderServiceRows(services []types.ServiceConfig, width, rowBudget int) string {
    if len(services) == 0 {
        return lipgloss.NewStyle().
            Foreground(appstyles.Active.TextDim).
            Background(appstyles.Active.BackgroundRecessed).
            Width(width).
            Render("No standalone services")
    }

    // Column widths for the overview: dot + name + image + state.
    // Simpler than the full member table — health/uptime/ports are
    // Services-page concerns.
    cols := overviewCols(width)

    show := services
    truncated := false
    if len(show) > rowBudget {
        show = show[:rowBudget-1]
        truncated = true
    }

    var rows []string
    for _, svc := range show {
        rows = append(rows, m.renderOverviewRow(cols, width, svc))
    }
    if truncated {
        dimStyle := lipgloss.NewStyle().
            Foreground(appstyles.Active.TextDim).
            Background(appstyles.Active.BackgroundRecessed).
            Width(width)
        rows = append(rows, dimStyle.Render(
            fmt.Sprintf("  … and %d more", len(services)-len(show))))
    }

    cardBg := appstyles.Active.BackgroundRecessed
    return appstyles.FillBackground(cardBg,
        lipgloss.JoinVertical(lipgloss.Left, rows...))
}
```

#### 1e. New helper: `overviewCols` and `renderOverviewRow()`

Simplified column layout and row renderer for the 4-column overview
(dot, name, image, state). Reuses the existing `truncate()` function,
`stateColor()`, and status dot colors from `GroupDetailsPanel`.

```go
type overviewCols struct {
    dot, name, image, state int
}

func overviewCols(width int) overviewCols {
    c := overviewCols{dot: 2, name: 20, image: 28, state: 10}
    total := c.dot + c.name + c.image + c.state
    if extra := width - total; extra > 0 {
        c.name += extra * 30 / 100
        c.image += extra * 50 / 100
        c.state += extra - extra*30/100 - extra*50/100
    }
    for c.dot+c.name+c.image+c.state > width {
        // Shrink the widest column.
        switch {
        case c.image > c.name && c.image > c.state:
            c.image--
        case c.name > c.state:
            c.name--
        default:
            c.state--
        }
    }
    return c
}

func (m GroupDetailsPanelModel) renderOverviewRow(cols overviewCols, width int, svc types.ServiceConfig) string {
    running := m.isServiceRunning(svc.Name)

    dotColor := appstyles.Active.StatusStopped
    if running {
        dotColor = appstyles.Active.StatusRunning
    }

    dot := "○"
    if running {
        dot = "●"
    }

    cardBg := appstyles.Active.BackgroundRecessed

    d := lipgloss.NewStyle().Foreground(dotColor).Background(cardBg).Width(cols.dot).Render(dot)
    n := lipgloss.NewStyle().Foreground(appstyles.Active.TextPrimary).Background(cardBg).Width(cols.name).Render(truncate(svc.Name, cols.name))
    i := lipgloss.NewStyle().Foreground(appstyles.Active.TextMuted).Background(cardBg).Width(cols.image).Render(truncate(svc.Image, cols.image))

    state := "stopped"
    if running {
        state = "running"
    }
    s := lipgloss.NewStyle().Foreground(stateColor(state)).Background(cardBg).Width(cols.state).Render(state)

    return lipgloss.NewStyle().Width(width).Background(cardBg).Render(
        lipgloss.JoinHorizontal(lipgloss.Left, d, n, i, s))
}
```

#### 1f. New helper: `pluralN()`

The existing `plural()` is unexported and lives in `GroupsList.go`. It takes
`(n int, word string)`. Either:
- Move it to a shared location (e.g., a `components/util.go`), or
- Duplicate it as `pluralN` in `GroupDetailsPanel.go`.

**Recommendation:** Extract `plural()` to a shared file since both components
use it. A one-line refactor, low risk.

#### 1g. Modify `renderBody()` — the only existing code that changes

The "no groups" branch in `renderBody()` is the single edit point:

```go
// BEFORE (current code):
if len(m.knownGroups()) == 0 {
    return renderEmptyCard(bodyWidth, bodyAvail, bg, "Getting started",
        "Groups are Compose profiles: sets of services you run together. Add a `profiles:` key to a service in your compose file to make one.",
        "n", "new group")
}

// AFTER:
if len(m.knownGroups()) == 0 {
    return m.renderServiceOverviewCard(bodyWidth, bodyAvail, bg)
}
```

That's the only change to an existing method. Everything else is new code.

#### 1h. Edge case: no services at all

If the compose file has zero services (unlikely but possible — an empty or
newly bootstrapped file), `renderServiceOverviewCard` should fall back to
the existing "Getting started" card. Add this guard at the top:

```go
func (m GroupDetailsPanelModel) renderServiceOverviewCard(...) string {
    if len(m.services) == 0 {
        return renderEmptyCard(width, availHeight, bg, "Getting started",
            "Your compose file has no services yet. Add services to it, or create a new compose file.",
            "n", "new group")
    }
    // … normal rendering
}
```

### 2. `src/components/styles.go` — no changes

No new styles needed. The card uses the same `BackgroundRecessed` / `BorderCard`
combination as the existing empty card, and the dot/state colors from the member
table.

### 3. `src/keys/Keys.go` — no changes

The empty state is read-only. No new bindings, no context changes. `n` is
already live on the Groups list (even when empty — see `keys.Active()`:
`own := []key.Binding{List.New}` is always included for the list). The `2`
key is a page digit, already handled globally.

### 4. `src/model/AppModel.go` — no changes

- `configSyncCmds()` already broadcasts `SetServicesListMsg` to every page
  on config load and page switch.
- `GetRunningContainers` already runs on config load, and the panel already
  receives the result.
- `helpContext()` uses `len(m.allGroupNames()) == 0` for `ListEmpty` — still
  correct (the left panel list is still empty even though the right panel
  now shows services).

### 5. `src/cmds/` — no changes

No new message types. The panel reuses `SetServicesListMsg` and
`GetRunningContainersMsg` it already handles.

### 6. Tests

#### 6a. Component test: `src/components/GroupDetailsPanel_test.go` (new file)

```go
package components

import (
    "strings"
    "testing"

    "github.com/compose-spec/compose-go/v2/types"
    "github.com/filipemolina/cais/src/apptypes"
    "github.com/filipemolina/cais/src/cmds"
)

func TestServiceOverviewCardShowsServiceCount(t *testing.T) {
    m := GroupDetailsPanel().(GroupDetailsPanelModel)
    m.panelWidth = 60
    m.panelHeight = 24
    m.services = []types.ServiceConfig{
        {Name: "radarr", Image: "linuxserver/radarr", Profiles: nil},
        {Name: "sonarr", Image: "linuxserver/sonarr", Profiles: nil},
        {Name: "plex", Image: "plexinc/pms-docker", Profiles: nil},
    }
    // No containers → all stopped.

    body := m.renderBody()

    if !strings.Contains(body, "3 services") {
        t.Errorf("expected service count in overview card, got:\n%s", body)
    }
    if !strings.Contains(body, "no groups yet") {
        t.Errorf("expected 'no groups yet' in overview card")
    }
}

func TestServiceOverviewCardShowsRunningState(t *testing.T) {
    m := GroupDetailsPanel().(GroupDetailsPanelModel)
    m.panelWidth = 60
    m.panelHeight = 24
    m.services = []types.ServiceConfig{
        {Name: "web", Image: "nginx", Profiles: nil},
        {Name: "db", Image: "postgres", Profiles: nil},
    }
    m.containers = []apptypes.DockerContainer{
        {Service: "web", State: "running"},
    }

    body := m.renderBody()

    if !strings.Contains(body, "running") {
        t.Errorf("expected 'running' state for web service")
    }
    if !strings.Contains(body, "stopped") {
        t.Errorf("expected 'stopped' state for db service")
    }
}

func TestServiceOverviewCardHidesGroupedServices(t *testing.T) {
    m := GroupDetailsPanel().(GroupDetailsPanelModel)
    m.panelWidth = 60
    m.panelHeight = 24
    m.services = []types.ServiceConfig{
        {Name: "grouped", Image: "img", Profiles: []string{"media"}},
        {Name: "standalone", Image: "img", Profiles: nil},
    }
    // Groups exist, so this test actually hits the "groups exist, none
    // selected" path — the overview card only shows when knownGroups()
    // is empty. So this test verifies the card is NOT shown.
    body := m.renderBody()

    if strings.Contains(body, "no groups yet") {
        t.Errorf("overview card should not show when groups exist")
    }
}

func TestServiceOverviewCardTruncatesLongLists(t *testing.T) {
    m := GroupDetailsPanel().(GroupDetailsPanelModel)
    m.panelWidth = 60
    m.panelHeight = 14 // tight vertical space
    // 10 services, only room for ~4 rows.
    for i := range 10 {
        m.services = append(m.services, types.ServiceConfig{
            Name:  fmt.Sprintf("svc-%d", i),
            Image: "img",
        })
    }

    body := m.renderBody()

    if !strings.Contains(body, "more") {
        t.Errorf("expected truncation indicator for long service lists")
    }
}

func TestServiceOverviewCardEmptyComposeFile(t *testing.T) {
    m := GroupDetailsPanel().(GroupDetailsPanelModel)
    m.panelWidth = 60
    m.panelHeight = 24
    // No services at all.

    body := m.renderBody()

    // Should fall back to the "Getting started" card rather than showing
    // an empty overview.
    if !strings.Contains(body, "Getting started") {
        t.Errorf("expected fallback card for empty compose file")
    }
}
```

#### 6b. Background-bleed test addition

Add a new case to `TestNoBackgroundBleedAcrossPages` in
`src/model/background_test.go`:

```go
{
    // The service overview card uses BackgroundRecessed with a border,
    // same as the old empty card. The row list inside it is the new
    // surface that could leak.
    name: "home service overview card",
    msgs: []tea.Msg{
        cmds.GetConfigMsg{
            FileName: "compose.yaml",
            Project: &types.Project{
                Services: types.Services{
                    "web":  {Name: "web", Image: "nginx"},
                    "db":   {Name: "db", Image: "postgres"},
                    "app":  {Name: "app", Image: "node"},
                },
            },
        },
    },
    width: 120,
},
```

This runs under `forEachTheme`, so both `stitcher-dark` and `stitcher-light`
are covered.

#### 6c. E2E rig test (optional, low priority)

A rig test that loads a project with services but no profiles and asserts
the overview card renders:

```go
func TestRigServiceOverviewCard(t *testing.T) {
    dir := t.TempDir()
    os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(`services:
  web:
    image: nginx:alpine
  db:
    image: postgres:16
`), 0o644)
    t.Chdir(dir)

    r := newRig(t)
    if !r.WaitFor("no groups yet", 3*time.Second) {
        t.Fatalf("service overview card never rendered. Output:\n%s", r.Output())
    }
    if !r.WaitFor("web", 3*time.Second) {
        t.Fatalf("service names not shown in overview. Output:\n%s", r.Output())
    }
}
```

### 7. Documentation updates

#### 7a. `README.md`

Update the "UI overview" section. The paragraph about Home currently says:

> **No groups yet:** a *Getting started* card explaining that groups are
> Compose profiles and how to create one.

Replace with:

> **No groups yet, services exist:** a service overview card listing every
> service in the compose file with its running state, and pointers to
> create the first group (`n`) or browse services (`2`).
>
> **No groups and no services:** a *Getting started* card explaining how
> to add services to the compose file.

#### 7b. `docs/DESIGN.md`

Update the "Home layout" section's bullet about the empty state:

> When no groups exist it shows an onboarding card

→

> When no groups exist but services do, it shows a service overview card
> listing each service's name, image, and running state, pointing the user
> at `n` (create group) and `2` (Services page). When the compose file is
> empty, it shows the original onboarding card.

#### 7c. `TODO.md`

Add a completed entry:

```markdown
- [x] **[S] Service-aware empty state** — when no groups exist but the
  compose file has services, the Group Details panel shows a read-only
  overview of every service (name, image, running state) instead of the
  static "Getting started" card. `n` creates the first group through the
  existing modal flow; `2` navigates to the Services page for full
  details. The card uses the existing `BackgroundRecessed` / `BorderCard`
  surface and the member-table dot/state colors, so it is visually
  consistent with the populated group view it transitions into.
```

## Implementation order

Each step compiles and passes tests on its own.

| Step | File | What | Risk |
|------|------|------|------|
| 1 | `src/components/GroupDetailsPanel.go` | Extract `plural()` to shared `components/util.go` | Trivial refactor |
| 2 | `src/components/GroupDetailsPanel.go` | Add `standaloneServices()` helper | New code, no callers yet |
| 3 | `src/components/GroupDetailsPanel.go` | Add `overviewCols` + `renderOverviewRow()` | New code, no callers yet |
| 4 | `src/components/GroupDetailsPanel.go` | Add `renderServiceRows()` | New code, no callers yet |
| 5 | `src/components/GroupDetailsPanel.go` | Add `renderServiceOverviewCard()` | New code, no callers yet |
| 6 | `src/components/GroupDetailsPanel.go` | **Wire it up:** change `renderBody()`'s "no groups" branch | The one edit to existing code |
| 7 | `src/components/GroupDetailsPanel_test.go` | Component tests | New file |
| 8 | `src/model/background_test.go` | Add "home service overview card" test case | One new case |
| 9 | Manual verification | `make dev` with a compose file that has no profiles | Visual check |
| 10 | Docs | Update README, DESIGN.md, TODO.md | Prose only |

## Acceptance criteria

1. Open Cais with a compose file that has services but no
   `profiles:` tags → the right panel shows the service overview card with
   every service listed by name, image, and running state.
2. Open Cais with a compose file that has groups → the right panel
   shows the existing "Select a group" prompt (no change).
3. Open Cais with an empty compose file → the right panel shows the
   "Getting started" card (fallback).
4. Press `n` from any of the above states → the existing create-group modal
   flow opens (name prompt → service checklist).
5. `go build ./... && go vet ./... && go test ./... && gofmt -l .` all green.
6. `go test -race ./...` green.
7. Background-bleed tests pass under both `stitcher-dark` and `stitcher-light`.
