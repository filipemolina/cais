# Design

The guiding principles of stack-stitcher. Written down so future
contributors (human or AI) have a north star when deciding where a
new feature belongs, what to call it, and how to think about the
data model.

## 1. The groups-first principle

**The home page operates on groups of services. The Services page operates on
individual services. This is a navigation rule, not just a feature.**

A *group* in stack-stitcher is a set of services that share a Compose
`profiles:` tag in the user's `compose.yml`. The user starts, stops, and
otherwise acts on *groups* from the home page — never on individual
services. The Services page exists for the rare case where you need to act on
one service in isolation.

When in doubt about which page a feature belongs on, ask:

- "Is this about a group of services, or a single service?"
  Group → home. Service → Services.
- "Does the user pick a thing, then act on a whole set?"
  Group. Home.
- "Does the user need to see the inner workings of one service?"
  Service. The Services page.

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
use the Services page (or a future dedicated page).

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
3. **Footer** — the `KeybindingBar`, which shows selection-aware hints (action
   keys are hidden when no group or service is selected, and list-empty keys
   are suppressed). It does not decide that itself: it tracks the state and
   asks `keys.Active` — see *Where keybindings live* below. The global keys
   (page digits, quit) sit on the right, apart from the context-dependent ones,
   with the resolved compose file dimmed just left of them — see *Which compose
   file* below.

### Navigation and focus

**Pages are switched with digits; the nav is not focusable.** `1` opens the
first tab, `2` the second, and so on; `[` and `]` step through the tabs in
order, wrapping around. The nav renders each tab's digit before its label (`1
Groups`), so the tab advertises its own key. `alt`+letter remains as an alias,
the letter derived from the label by `apptypes.PageShortcut`; it is advertised
only in the `?` overlay.

Why digits at all: `alt` was the weakest link in the old scheme. macOS
Terminal.app and iTerm2 do not send Option as Alt until the user changes a
setting, so `alt+g` silently did nothing for part of the audience. Digits need
no modifier and no terminal cooperation. They also drop the constraint that no
two labels may share a first letter, which the underline scheme needed; a
shared letter now just means the alias resolves to the first matching page.
Even the alias avoids `ctrl`: terminals intercept `ctrl+s` as XOFF flow
control and `ctrl+d` as end-of-input.

All page keys are handled in `AppModel.Update`, after the modal check and
inside the `keyboardOwned()` guard, so typing in a text field or a list filter
can never navigate away - while a filter is being typed, `1` is a letter. The
brackets are declared in `src/keys`; the digits are matched by key code in
`pageForNavKey`, because which digits are live depends on how many pages there
are - and the footer's `1-N` hint is derived from that same list, so a new tab
extends both instead of drifting from them.

**`esc` is "back", as a ladder of claims.** Strongest first: a modal closes
itself; a filter being typed owns the keyboard and esc abandons it; a focused
list holding an applied filter keeps esc, because esc is the only way back to
the full rows - the list says so through `KeepsEsc()`, asked via
`AppModel.escKept()` the same way `OwnsKeyboard()` is asked, because the
answer has to be right on the keystroke that changes it. What remains is the
details panel, where esc returns focus to the list. When a filter stands on an
*unfocused* list, esc moves focus to the list first and clears the filter on
the next press, so the user is never stranded in a filtered list with no
advertised way out. The footer offers `esc back` in the details contexts only:
everywhere else the key is either spoken for or does nothing, and the bar does
not advertise inert keys.

`tab` still does nothing while a filter is being typed: the filter input owns
the keyboard, and `enter`/`esc` are the only ways out of it. Making `tab`
apply-and-move would resurrect the one-key-two-jobs collision the list keymap
work removed, so inert it stays.

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

### Where keybindings live

**Every key is declared exactly once, in `src/keys`.** A key used to live in two
places — the component that handled it, and the hand-written list in
`KeybindingBar` that advertised it — with nothing holding the two together, so
the bar could promise a key no handler implemented, or stay silent about one
that did. Components now match with `key.Matches(msg, keys.Details.Start)`, and
the footer renders the help text the binding itself carries.

Two rules follow, and they are the reason the package exists:

1. **One verb is one binding.** `s` starts a group and starts a service because
   both panels read `keys.Details.Start` — not because two switch statements
   happen to agree. The shared docker verbs resolve through
   `components.dockerActionFor`.
2. **The footer asks the keymap; it does not keep a list.**
   `keys.Active(keys.Context{…})` takes the page, the focused component, whether
   the list is empty and whether anything is selected, and returns the bindings
   that are live, in display order. The bar supplies the screen state, the keymap
   makes the decision, so the two cannot disagree.
   `components.TestFooterHints` pins every context.

`Active` returns a *filtered slice* rather than calling `SetEnabled` on the
bindings it wants to hide. `key.Binding.Enabled` gates matching as well as help,
and these are package-level values shared with the components: disabling one to
tidy the footer would stop the key working everywhere.

The help overlay is the package's third reader, after the components and the
footer. `?` opens it through `cmds.OpenHelpModal` — the same message path every
modal takes — and it renders `keys.Catalog(ctx)`: every binding grouped by
scope, with availability resolved against a snapshot of the screen it opened
from (`AppModel.helpContext`: page, focus, selection, and the list's filter
state through a small `filterStater` interface). A row that does nothing on
that screen is dimmed, and the snapshot cannot go stale because a modal freezes
the panels beneath it. It closes with `?`, `esc` or `q` — `q` closes only the
overlay — and, being an overlay with the footer hidden beneath it, it
advertises those close keys itself, built from the same bindings. It is also
where the keys with no footer room live: the `alt`+letter aliases (one derived
row), the `[`/`]` brackets, `g`/`G`, `shift+tab` and `ctrl+c`.

The tiers:

| Tier | Keys | Rule |
| --- | --- | --- |
| Global | digits, `[` / `]`, `?`, `esc` (back), `tab` / `shift+tab`, `q` | Same meaning everywhere, never contextual — but they yield to whatever owns the keyboard (see the esc ladder under *Navigation and focus*) |
| Force quit | `ctrl+c` | Yields to nothing, checked before the modal handoff |
| Panel | lowercase letters (`s t r p x l e n d`) | Act on the focused panel's selection; one verb, one key, on every panel |
| Destructive | `x`, `d` | Always through `ConfirmModal`; never dispatched straight from a panel |
| Overlay | `esc` cancel, `enter` confirm, `y` / `n`, plus overlay-local letters | The overlay owns the keyboard while it is open |
| List | cursor keys, `g` / `G`, `/` | The list's own, and the only keys `list.KeyMap` is allowed to claim |

**There is no prefix key**, and this is deliberate. Prefixes (tmux `ctrl+b`,
zellij `ctrl+p`) exist because those programs host another program that owns the
keyboard; the prefix is how you address the host without stealing keys from the
guest. Stack Stitcher has no guest — the only things needing raw keys are text
inputs, and those live in modals that capture everything. A prefix would add a
mode to teach, render and exit, and would resolve no conflict. The comparable
tools (lazydocker, lazygit, k9s) don't use one either: they use one global tier,
one panel-scoped tier, and a `?` overlay listing both.

Because an overlay hides the footer bar while it is open, **an overlay advertises
its own keys**. It builds that line from the same bindings, through
`components.renderKeyHints`, so a modal's help and the footer read alike; pass a
lighter description color when the modal sits on a lighter surface than the bar.

#### The lists do not get to keep `list.DefaultKeyMap`

A bubbles `list.Model` installs `list.DefaultKeyMap()`, which is written for a
list that *is* the whole program. It binds `d` and `f` to next-page, `h`, `b` and
`u` to previous-page, and takes `q`, `esc` and `?` for itself. Both body lists
hand every key to the inner list while focused, *after* matching their own, so
those keys did two jobs at once: `d` opened the delete-group confirm **and**
paged the list out from under it.

So the lists install `keys.ListKeyMap()` instead. It keeps only what the list
alone can answer — cursor movement, `g`/`G`, and `/` — and leaves every key the
app owns bound to nothing. `components.TestDeleteKeyDoesNotAlsoPageTheList` and
`TestPanelLettersDoNotPageTheList` fail against the default map.

**A list being filtered is an overlay.** While a filter is being typed the
keystrokes are text: `n` is not "new group", `q` is not "quit". The list says so
through `OwnsKeyboard()`, `AppModel.keyboardOwned()` asks every component on the
active page, and `Update` drops out of its own key handling when the answer is
yes — dropping out rather than returning, because the component below still needs
the keystroke for the filter input. That is an interface rather than a broadcast
because the answer has to be right on the very keystroke that changes it.

Two consequences worth keeping:

- **`ctrl+c` is its own binding** (`keys.Global.ForceQuit`), matched before the
  modal handoff, so it quits whatever owns the keyboard. `q` is the one that
  yields. Previously the two shared a binding and `ctrl+c` did not quit while a
  modal was open.
- **The footer follows the filter.** The lists broadcast
  `cmds.SetListFilterStateMsg` on the transition, so filtering advertises only
  the keys that end it, and an applied filter turns the `/` slot into the `esc`
  that clears it. Otherwise the bar would go on offering `n`, `d` and `space`
  while all three were letters — the drift `src/keys` exists to prevent.
  `esc` is also the reason the list keeps a key the app otherwise owns: without
  it an applied filter has no way out.

### Which compose file

**The file priority order is fixed, matches Docker's, and is not a setting.**
`utils.GetComposeFileName` tries `compose.yaml`, `compose.yml`,
`docker-compose.yaml`, `docker-compose.yml` in that order — the same order
Docker uses, because `utils.DockerCompose`, `utils.DockerComposePs` and
`utils.DockerLogs` all shell out to `docker compose …` **without `-f`** and let
Docker resolve the file itself. The TUI and the commands it runs agree today by
duplicating that order. Making it configurable would break that agreement: the
panel would describe `compose.yaml` while `docker compose start` acted on
`docker-compose.yml`.

What was missing is not a setting but an answer to "which file?", so
`AppModel` broadcasts the name it resolved as `cmds.SetComposeFileMsg` and the
footer reports it. It degrades as the terminal narrows — full path, then
basename, then dropped — because the keys beside it are worth more than the
name; `components.TestFooterComposeFile` pins that ladder, and
`TestFooterComposeFileNeverCrowdsOutTheKeys` pins the part that matters, that
adding the name never costs the bar a line or a key. When several candidates
exist the name alone is only half the answer: `utils.GetComposeFileName`
returns every candidate in priority order, the footer marks the winner with
`+N` (the count rides through the same ladder), and the help overlay lists
the losers by name.

**This constrains the planned `-f`/`--file` flag.** The flag is only safe once
every docker invocation passes the resolved path; threading it through those
three `utils` call sites is a prerequisite of the flag, not a follow-up.

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

### Editing services

Service editing hands the user their own YAML rather than a form. A form has
to pick a representation for every field, and compose accepts two spellings
for both `ports:` (short string vs. long mapping) and `environment:` (list
vs. mapping), so round-tripping through inputs would silently rewrite
whichever the user chose along with their comments and key order. A form is
also a standing tax: any field nobody modelled is a field nobody can edit.

`e` extracts one service as a single-key mapping, opens it in `$VISUAL`/
`$EDITOR`, and splices the result back over the same key. `E` opens the
whole file, which is the only way to add a service or touch top-level keys.

Nothing is written unless the fragment parses, keeps its name, and the whole
resulting document still loads as compose — validated by writing a candidate
next to the compose file and running the loader over it. A rejected edit
reports the loader's message and returns to a normal TUI with the file
untouched; pressing the key again is the retry. Do not add a
fix-it-before-you-can-leave loop: being unable to leave is worse than losing
the text.

Renaming a service through the fragment is refused, because `depends_on:`
elsewhere in the file may point at the old name.

Every write re-encodes through `yaml.v3`, which round-trips comments but not
blank lines, so the spacing between services is lost. This is accepted rather
than worked around. The obvious trick - carrying blank lines through as
marker comments - was built and then removed: a blank line inside a block
scalar (`command: |`) is part of the string, so the transformation has to
know where it must not apply, and quietly rewriting the user's data is a
worse failure than closing up their spacing.

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

### Color lives on a Theme

Every color the app draws with is a field on `appstyles.Theme`
(`src/appstyles/Theme.go`), not a hex value scattered through a component.
`appstyles.Active` is the one `Theme` in effect; every call site reads it
fresh - `appstyles.Active.TextPrimary`, say - rather than caching a color at
package init, which is what lets a later switch actually repaint: assign a
different registered `Theme` to `Active` and the next frame draws it. The
handful of styles that are more than one field - `appstyles.NormalTitle`,
`LogsModal.go`'s `logsModalWrapper`, `model/View.go`'s `errorBannerStyle` -
are functions for the same reason, not package-level `var`s: a `var` built at
init freezes whichever theme was active when the package loaded, and that
used to be the whole palette's problem before this existed.

`appstyles.Themes` is the registry (`stitcher-dark`, the default, and
`stitcher-light`), built by `appstyles.newTheme` from a handful of base
colors - `Accent`, the text/panel/modal bases, `Danger`, the four status
colors - with everything else derived by `Lighten`/`Darken`. A dark theme
raises a tier's attention by lightening it, a light theme by darkening it;
see the constructor's doc comment for why both directions use the same
deltas. Adding a theme is choosing those base colors, not hand-tuning thirty
derived ones. There is no switcher UI yet (post-alpha, see
`docs/ROADMAP.md`); `Active` is the seam it will use.

Two fields are the one deliberate exception to "derived from base colors":
`InkOnLight`/`InkOnDark` do not vary with a theme's `Dark` flag, because a
status pill's own fill (`StatusRunning` green, `StatusStarting` amber,
`StatusError` red) does not vary with the app's theme either - the text that
reads legibly on a green pill has to stay dark whichever theme is active, not
follow `TextPrimary`, which flips. `GroupDetailsPanel.go`'s `statusPill` used
to reach for `PanelBg`/`TextPrimary` as stand-ins for "a dark color" and "a
light color"; that only ever worked because the one theme that existed was
dark, and `stitcher-light` is exactly what exposed it.

### Background tiers, and sealing them

Sections are separated by background color rather than by borders. The tiers
are `Theme` fields (`src/appstyles/Theme.go`), read through `appstyles.Active`:

| Tier | Field                | Where                                  |
| ---- | -------------------- | -------------------------------------- |
| 1    | terminal default     | outside the app — never drawn on        |
| 2    | `BackgroundContent`  | the frame: header, footer, gutter       |
| 3    | `BackgroundPanel`    | the body panels                        |
| 4    | `BackgroundElevated` | the focused panel                      |
| —    | `ModalBg`            | modals, and an active list row — its own register, not derived from the panel tiers |

Focus is shown by lifting a panel from tier 3 to tier 4, not by a heavier
border, so a panel's box is the same size whether or not it is focused. Use
`components.panelBg(isFocused)` rather than repeating that choice.

One surface runs the other way. `BackgroundRecessed` sits *below* the panel
tier — it is the theme's un-raised `PanelBg` — and is used for insets like the
empty-state cards, which read as cut into the panel rather than raised off it.
Because it is the theme's own base rather than a raise of it, it stays at the
ladder's base end regardless of which direction raising goes for the active
theme.

A recessed surface needs `BorderCard`, not `BorderDefault`: a rim has to
stand out against the surface it wraps. `BorderDefault` moves toward the base
end with `BackgroundRecessed` rather than away from it, so it all but
disappears against a recessed fill; `BorderCard` moves the other way, toward
more contrast, which is why it is the one that rims a recessed surface
instead.

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
both pages and their empty, populated, narrow and error-banner states -
now once per registered theme, via a `forEachTheme` helper that sets
`appstyles.Active` and restores it after. A theme that leaves a field
zero-valued (a nil `color.Color` sets no SGR at all) fails the same suite
every other state already runs, rather than shipping unnoticed; see
`src/appstyles/Theme_test.go` for the same property one level down, on the
fields themselves rather than a rendered frame.

## 6. Decision checklist for new features

Before adding a feature, answer these:

1. **Group or service?** See §1. If neither, it probably belongs on a new
   page (e.g. monitoring, settings).
2. **Can it be expressed as `profiles:` tag manipulation?** See §3. If yes,
   it fits the home page's mental model. If no, it may need a different
   mechanism.
3. **Does it require editing existing services?** That capability is
   currently a roadmap item. The first place to add it is the Services page,
   not home — home is for groups.
4. **Does it conflict with the Services page's role?** If a feature would
   duplicate Services-page functionality, prefer to extend that page unless
   there's a clear group-level reason.

## 7. Related documents

- [Roadmap](ROADMAP.md) — the ordered plan to a first alpha, the decisions
  already taken with the owner, and which phase is next. Live.
- [Current TODO](../TODO.md) — the live worklist and recent completed work.
- [Create/delete profiles design](superpowers/specs/2026-07-22-create-delete-profiles-design.md) —
  completed historical design for the create/delete-groups flow.
- [Create/delete profiles plan](superpowers/plans/2026-07-22-create-delete-profiles.md) —
  completed historical implementation plan for that flow.
- [Bootstrap compose file design](superpowers/specs/2026-07-23-bootstrap-compose-file-design.md) —
  completed historical design for bootstrapping a compose file from inside the TUI.
- [Bootstrap compose file plan](superpowers/plans/2026-07-23-bootstrap-compose-file.md) —
  completed historical implementation plan for that flow.
