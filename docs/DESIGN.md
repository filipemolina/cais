# Design

The guiding principles of stack-stitcher. Written down so future
contributors (human or AI) have a north star when deciding where a
new feature belongs, what to call it, and how to think about the
data model.

## 1. The groups-first principle

**The home page operates on groups of services. The dashboard operates on
individual services. This is a navigation rule, not just a feature.**

A *group* in stack-stitcher is a set of services that share a Compose
`profiles:` tag in the user's `compose.yml`. The user starts, stops, and
otherwise acts on *groups* from the home page — never on individual
services. The dashboard exists for the rare case where you need to act on
one service in isolation.

When in doubt about which page a feature belongs on, ask:

- "Is this about a group of services, or a single service?"
  Group → home. Service → dashboard.
- "Does the user pick a thing, then act on a whole set?"
  Group. Home.
- "Does the user need to see the inner workings of one service?"
  Service. Dashboard.

## 2. Terminology: group vs profile

We use **group** in the UI and documentation. The underlying Compose YAML
field is still `profiles:` because that's what Docker calls it.

| We say                    | Compose says                            |
| ------------------------- | --------------------------------------- |
| group                     | profile                                 |
| "Create a group"          | adds a new `profiles:` tag to services  |
| "Start a group"           | `docker compose --profile <name> up`     |

The data layer still references `service.Profiles` (the field name from
[`compose-go`](https://github.com/compose-spec/compose-go)). That's an
implementation detail; the user-facing word is **group**.

## 3. The data consequence

A group is **not** a first-class object in `compose.yml`. It only exists as
a `profiles:` string tag on individual services. stack-stitcher derives the
visible group list by scanning every service's `Profiles` field
(`allGroupNames()` in `src/model/AppModel.go`).

This means:

- **Creating a group** = tagging one or more services with a new profile name.
- **Deleting a group** = stripping that tag from all services that have it.
- **A group can become empty** if its last service is removed or untagged.
- **Renaming a group** is currently not supported (would require multi-file
  YAML rewriting).

This is a deliberate constraint: we don't add a separate "groups" file or a
sidecar index. The user's `compose.yml` is the source of truth.

## 4. What home is not

Home is **not** a dashboard. It is not a place to monitor metrics, see CPU
graphs, or get notifications. It is a launchpad: pick a group, do an action
on the whole group, move on.

For per-service introspection, monitoring, or editing existing services,
use the dashboard (or a future dedicated page).

## 5. Layout and navigation contract

The UI is a full-height TUI. The terminal is divided into three stacked
regions:

1. **Header** — the page tabs on the left, the wordmark (`▌ Stack Stitcher`)
   pinned to the right. Tabs are decoupled from page IDs via
   `apptypes.PageLabels`: *Home* is displayed as **Groups**, *Compose Files* as
   **Files**.
2. **Body** — the remaining rows after the header and footer. The exact box
   for each of the two panels is computed by `AppModel`
   (`calculateBodyLayout`) and broadcast as `cmds.SetBodyLayoutMsg` on every
   `WindowSizeMsg`, on every `SetActivePageMsg`, and whenever the error
   banner appears/disappears.

   **`AppModel` is the only place that reads the terminal dimensions.**
   Components size themselves from the broadcast box and never derive
   width or height from `WindowSizeMsg`: that message only reaches the
   components of the page that is active when it arrives, so a page that
   was not active during a resize — including *every* page at startup,
   since no page is active when the terminal is first measured — would
   render at width 0.

   The layout guarantees
   `LeftWidth + BODY_GUTTER_WIDTH + RightWidth == terminal width`. The left
   panel takes `LEFT_PANEL_WIDTH` of the row remaining after the gutter and
   the right panel takes what is left, so rounding can never overflow the
   row or leave a ragged column. Panels render at exactly that box —
   `Width`/`Height` to fill it and `MaxWidth`/`MaxHeight` to clip, because
   lipgloss `Width()` pads but does not truncate, and anything wider than
   the terminal gets wrapped by the terminal itself.
3. **Footer** — the `KeybindingBar`, which is updated by `AppModel` and
   shows selection-aware hints (action keys are hidden when no group or
   service is selected, and list-empty keys are suppressed). The global keys
   (page chords, quit) sit on the right, apart from the context-dependent ones.

### Navigation and focus

**Pages are switched with `alt`+letter; the nav is not focusable.** The letter
is the first letter of the tab's label, which the nav underlines, so the tab
advertises its own shortcut. `apptypes.PageShortcut` derives the letter from the
label rather than reading a table, so the underline and the binding cannot drift
apart; `TestPageShortcutsAreUnique` guards against two labels sharing a letter.

`alt` and not `ctrl`: terminals intercept `ctrl+s` as XOFF flow control and
`ctrl+d` as end-of-input, so those chords are unreliable on two of the four
pages. The chords are handled in `AppModel.Update`, after the modal check, so
typing in a text field can never navigate away.

Because the nav takes no focus, `constants.FocusableComponents` holds only the
two body panels, and `Tab`/`Shift+Tab` alternate between them. Component ids are
part of the focus protocol (a component compares its id against
`cmds.SetFocusMsg`), so they are *not* positions in the focus cycle — the nav is
id 0 but is absent from the cycle. `ChangeFocus` derives the cycle position from
the currently focused id so the two cannot disagree.

**Every page in `apptypes.PageTitles` needs an entry in `AppModel.pages`.** The
map drives rendering, the layout broadcast and the focus cycle. A page listed in
the nav but missing from the map renders an empty body; `View` guards that case
by always setting `AltScreen`, because returning the zero `tea.View` drops the
terminal out of the alternate screen and the app looks like it crashed while
still running. Pages that aren't implemented yet get a
`components.PlaceholderPanel`.

### State refresh and destructive actions

`AppModel` owns container-status refreshes as well as terminal layout. A
successful config load queues a foreground `GetRunningContainers` refresh, and
page changes refresh only after a project has loaded. This avoids letting a
failed `docker compose ps` overwrite the useful bootstrap error in an empty
directory.

`cmds.RefreshContainersTick` re-schedules a five-second poll for the life of
the app. It dispatches `GetRunningContainersBackground` only while a project is
loaded and no modal is open. Background results update status without clearing
an unrelated action/configuration error; a recovered background poll clears its
own error banner. Keep this distinction if the refresh mechanism changes.

`x` removes containers (`docker compose rm -fs`) and is therefore destructive.
It must go through `cmds.OpenConfirmModal` / `ConfirmModal`; `y` runs the
follow-up command and `n` or `Esc` cancels it. Do not dispatch a remove action
directly from a panel.

`DockerComposePs` invokes `docker compose ps --format json` directly. Its
parser accepts both a JSON array and legacy NDJSON, so `jq` is deliberately not
a runtime dependency.

The compose file is the user's own, and it is the one piece of state the app
cannot recreate. Every write to it goes through `utils.ReplaceFileAtomically`,
which writes a temporary file alongside the target and renames it into place,
carrying the original's permissions across. Nothing may write to a compose file
with `os.WriteFile` or an `os.Create` truncation: those destroy the original
before the new contents are safely on disk, so a failure mid-write leaves the
user with nothing. Encode into memory first, as `writeComposeNode` does, so
that a serialisation error never reaches the file at all.

### Home layout

Home is the launchpad. Its body is a two-pane layout:

- **Groups list** — the selectable list of derived groups (Compose profiles)
  with a status header. The empty state is rendered as normal panel text, not
  an inverted box.
- **Group Details** — the right panel. When no groups exist it shows an
  onboarding card; when groups exist but none is selected it prompts the user
  to pick one; when a group is selected it shows a header card with a status
  pill, a running/stopped/services summary, a member-services table (status
  dot, NAME, IMAGE, STATE, HEALTH, UPTIME, PORTS), and pinned
  Start/Stop/Restart/Pull/Remove action buttons.

The large ASCII logo is no longer rendered here; it remains reserved for a
future About modal.

### Background tiers, and sealing them

Sections are separated by background color rather than by borders. The tiers
are defined in `src/appstyles/styles.go`:

| Tier | Token                | Where                                  |
| ---- | -------------------- | -------------------------------------- |
| 1    | terminal default     | outside the app — never drawn on        |
| 2    | `BackgroundContent`  | the frame: header, footer, gutter       |
| 3    | `BackgroundPanel`    | the body panels                        |
| 4    | `BackgroundElevated` | the focused panel, and modals           |

Focus is shown by lifting a panel from tier 3 to tier 4, not by a heavier
border, so a panel's box is the same size whether or not it is focused. Use
`components.panelBg(isFocused)` rather than repeating that choice.

One surface runs the other way. `BackgroundRecessed` sits *below* the panel
tier — it is the unlightened `PanelBg` — and is used for insets like the
empty-state cards, which read as cut into the panel rather than raised off it.
Because it is a fixed color rather than a lightening of its parent, it stays
darker than every tier above it, including a focused panel's tier 4.

A recessed surface needs `BorderCard`, not `BorderDefault`: a rim has to be
lighter than the surface it wraps, and `BorderDefault` is darker than
`PanelBg`, so it disappears against a recessed fill.

**Every tier must be sealed with `appstyles.FillBackground`.** A terminal's
SGR reset clears the background until the next SGR, and lipgloss closes each
styled run with a reset — so any unstyled text later on the same line renders
on the terminal's own color. `lipgloss.JoinVertical`/`JoinHorizontal` pad
shorter blocks out to their widest sibling with exactly that: bare spaces
carrying no SGR. Several bubbles components join their rows the same way.
Wrapping the result in a `Background()` style does not help, because a style
only paints the padding it adds itself.

Two rules follow:

1. **Anything that draws text needs an explicit background**, including
   buttons, cards and list rows. A run with no background set is the notch.
   Components that sit inside a panel take that panel's tier as a parameter
   instead of picking a tint of their own, so they stay flush when focus
   lifts the panel.
2. **Seal innermost-first.** Each tier seals its own region, then the next
   tier out seals what is left. Sealing only at the outer tier would flatten
   the inner ones — the active list row's lighter surface would be repainted
   to the panel color. The outermost seal is the tier-2 pass in
   `AppModel.View`.

`appstyles.HasBackgroundBleed` is the matching assertion, and
`src/model/background_test.go` applies it to fully rendered frames across
both pages and their empty, populated, narrow and error-banner states. A new
component that joins blocks without sealing them fails there.

## 6. Decision checklist for new features

Before adding a feature, answer these:

1. **Group or service?** See §1. If neither, it probably belongs on a new
   page (e.g. monitoring, settings).
2. **Can it be expressed as `profiles:` tag manipulation?** See §3. If yes,
   it fits the home page's mental model. If no, it may need a different
   mechanism.
3. **Does it require editing existing services?** That capability is
   currently a roadmap item. The first place to add it is the dashboard,
   not home — home is for groups.
4. **Does it conflict with the dashboard's role?** If a feature would
   duplicate dashboard functionality, prefer to extend the dashboard unless
   there's a clear group-level reason.

## 7. Related documents

- [Current TODO](../TODO.md) — the live worklist and recent completed work.
- [Create/delete profiles design](superpowers/specs/2026-07-22-create-delete-profiles-design.md) —
  completed historical design for the create/delete-groups flow.
- [Create/delete profiles plan](superpowers/plans/2026-07-22-create-delete-profiles.md) —
  completed historical implementation plan for that flow.
- [Bootstrap compose file design](superpowers/specs/2026-07-23-bootstrap-compose-file-design.md) —
  completed historical design for bootstrapping a compose file from inside the TUI.
- [Bootstrap compose file plan](superpowers/plans/2026-07-23-bootstrap-compose-file.md) —
  completed historical implementation plan for that flow.
