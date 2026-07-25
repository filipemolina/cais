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

1. **Header** — the wordmark (`▌ Stack Stitcher`) plus page tabs. Tabs are
   decoupled from page IDs via `apptypes.PageLabels`: *Home* is displayed as
   **Groups**, *Compose Files* as **Files**.
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
   service is selected, and list-empty keys are suppressed).

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

- [Create/delete profiles design](superpowers/specs/2026-07-22-create-delete-profiles-design.md) —
  the design spec for the create/delete-groups flow.
- [Create/delete profiles plan](superpowers/plans/2026-07-22-create-delete-profiles.md) —
  the implementation plan for that flow.
- [Bootstrap compose file design](superpowers/specs/) — the design spec for
  bootstrapping a new compose file from inside the TUI.
