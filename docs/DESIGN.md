# Design

The guiding principles of Cais. Written down so future
contributors (human or AI) have a north star when deciding where a
new feature belongs, what to call it, and how to think about the
data model.

## 1. The groups-first principle

**The home page operates on groups of services. The Services page operates on
individual services. This is a navigation rule, not just a feature.**

A *group* in Cais is a set of services that share a Compose
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
a `profiles:` string tag on individual services. Cais derives the
visible group list by scanning every service's `Profiles` field
(`allGroupNames()` in `src/model/AppModel.go`).

This means:

- **Creating a group** = tagging one or more services with a new profile name.
- **Deleting a group** = stripping that tag from all services that have it.
- **Editing a group's membership** = reconciling tags so exactly the chosen
  services carry it (see *Editing group membership* below).
- **A group can become empty** if its last service is removed or untagged.
- **Renaming a group** is a value rename: replace the tag wherever it
  appears in the loaded compose file (`utils.RenameGroupTag`). Nothing else
  in a compose file references a profile by name (unlike service names,
  which `depends_on:` references), so a rename cannot leave dangling
  references the way a service rename would. The rename is scoped to the
  file the app has loaded; a service whose tag lives in *another* file of a
  multi-file project keeps the old name (the app loads exactly one file —
  see *Which compose file*).

This is a deliberate constraint: we don't add a separate "groups" file or a
sidecar index. The user's `compose.yml` is the source of truth.

**An untagged service is not "no group" — it is every group.** Docker
Compose starts a service with no `profiles:` key alongside *any* profile
that is requested, cais's included: `docker compose --profile core up -d`
also brings up every service that carries no tag at all, the same as it
would for `--profile anything-else`. Cais's groups list only shows what has
a tag, so an untagged service is invisible there while still riding along
on every group's start - and if it is broken (bad image, bad config), it
can fail *every* group's start without ever being the group the user
picked. This is Compose's own semantics, not something cais can opt out of,
so the groups list's stats footer says how many ungrouped services exist
once at least one group does (`cmds.SetHomeStatsMsg.Ungrouped`,
`groupslist.statsLine`) - the standing count exists specifically so a
failed start does not have to be traced back to a service nobody selected.

## 4. What home is not

Home is **not** a dashboard. It is not a place to monitor metrics, see CPU
graphs, or get notifications. It is a launchpad: pick a group, do an action
on the whole group, move on.

For per-service introspection, monitoring, or editing existing services,
use the Services page (or a future dedicated page).

## 5. Layout and navigation contract

The UI is a full-height TUI. The terminal is divided into three stacked
regions:

1. **Header** — the page tabs on the left, the wordmark (`▌ Cais`)
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
`placeholderpanel.New`.

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

Each scope renders one key/description **per line** rather than packing
several side-by-side and wrapping — a scope with several keys used to read as
a run-on paragraph once it wrapped past one line. The full catalog is taller
than a terminal, so it windows and scrolls with `↑`/`↓`
(`keys.Overlay.Navigation`), and narrows with the same `/`-fuzzy filter every
list in the app already uses (`keys.List.Filter`), matching each entry's key
and description text. A scope with no matching entry disappears entirely
rather than leaving an empty heading, so filtering never breaks the section
separation described above; the compose-files section is not a keybinding
and only shows with an empty query.

`a` opens the About overlay: the figlet brand mark (`banner.Banner`,
gradient-colored from the active theme), the wordmark and slogan, the
version, the license and the repo link. It is a read-only surface like the
help overlay and closes on the same three keys (`a`, `esc`, `q`). `a` is
advertised in the help overlay rather than the footer — the footer is
width-constrained, and `?` is the
comprehensive list — so it lives in `pressableNow` (always available) and the
Catalog's Global scope, not in `Globals()` (the footer's pinned right side).

The tiers:

| Tier | Keys | Rule |
| --- | --- | --- |
| Global | digits, `[` / `]`, `?`, `a` (about), `esc` (back), `tab` / `shift+tab`, `q` | Same meaning everywhere, never contextual — but they yield to whatever owns the keyboard (see the esc ladder under *Navigation and focus*) |
| Force quit | `ctrl+c` | Yields to nothing, checked before the modal handoff |
| Panel | lowercase letters (`s t r p x l e n d h`) | Act on the focused panel's selection; one verb, one key, on every panel |
| Destructive | `x`, `d` | Always through `ConfirmModal`; never dispatched straight from a panel |
| Overlay | `esc` cancel, `enter` confirm, `y` / `n`, plus overlay-local letters | The overlay owns the keyboard while it is open |
| List | cursor keys, `g` / `G`, `/` | The list's own, and the only keys `list.KeyMap` is allowed to claim |

**There is no prefix key**, and this is deliberate. Prefixes (tmux `ctrl+b`,
zellij `ctrl+p`) exist because those programs host another program that owns the
keyboard; the prefix is how you address the host without stealing keys from the
guest. Cais has no guest — the only things needing raw keys are text
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
Docker uses. Making the *order* configurable would only mean answering "which
file?" differently from the tool the app is a front end for, in a way the user
would then have to keep in their head.

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

#### One resolution, passed down — never two

**The app resolves the file once and tells docker which one it picked.** Every
invocation starts from `utils.ComposeFileArgs`, which opens the argument list
with `compose --file <path>`, so `utils.DockerCompose`, `utils.DockerComposePs`
and `utils.DockerLogs` act on exactly the file the panels are describing.

It was not always so. Each side used to resolve independently — the app through
`GetComposeFileName`, docker through its own identical order — and they agreed
only because both were looking in the same directory for the same names. That
is a coincidence, not a guarantee, and `-f`/`--file` was the flag that would
have ended it: the footer would have named the file the user asked for while
`docker compose start` acted on whatever was in the working directory. So the
threading landed **first**, as its own change, and the flag after it. Anything
that later runs a docker command belongs on the same path; a second resolution
anywhere is the bug coming back.

The same argument applies to writes, which is why `cmds.CreateGroup` and
`cmds.DeleteGroup` take the file name as an argument instead of calling
`GetComposeFileName` again inside the command.

A component cannot supply the path, because a component does not know it and
should not: `AppModel` holds the resolved file, so panels emit an intent —
`cmds.RunDockerActionMsg`, `cmds.CreateGroupRequestMsg`, `cmds.OpenEditorMsg` —
and `AppModel` turns it into the command that carries the file.

#### `-f`/`--file` and `-d`/`--dir`

`utils.ComposeSource` carries whichever flag was given from `main.go` to the
resolver. `--dir` resolves in that directory in the usual order and returns
paths joined with it, so nothing downstream needs to know where they came from;
`--file` skips resolution altogether, because the user named the file and there
is nothing left to choose between — which also means no losing candidates for
the footer's `+N` to count.

They are refused together rather than layered. `--dir` says where to look and
`--file` says what to open, so honouring both would mean deciding which of two
answers the user meant, and either flag alone already covers the other's use.

Bad paths fail in `main.go`, before the alternate screen is entered. A typo in
the command the user just typed belongs in the shell they typed it into, not in
an error banner behind a full-screen app they have to quit to read.

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
own error banner. Esc is the manual dismissal that did not exist before: it is
the next rung in esc's priority ladder (after a modal closes, a filter being
typed owns the keyboard, and an applied filter keeps esc), so when no stronger
claim holds and a banner is showing, the first esc clears it and the next esc
backs out of the details panel. Keep this distinction if the refresh mechanism
changes.

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

`e` on the Services details panel opens the service in an inline `textarea`
that holds the same YAML fragment the external editor uses. `ctrl+s` saves
through `utils.ApplyServiceFragment`, `ctrl+o` opens the same fragment in
`$VISUAL`/`$EDITOR` for when the panel is too cramped, and `esc` cancels —
confirming first if the buffer has changed. The editor owns the keyboard
while it is open, so the docker action keys (`s`, `t`, `r`, `p`, `x`, `l`)
are plain text then. `E` still opens the whole compose file in `$EDITOR`,
which is the only way to touch top-level keys (`name:`, `volumes:`, …).

**Adding a service stays true to the same argument, one step earlier.** `n`
on the Services page opens a small two-field modal (Phase 1 of
`docs/plans/image-search.md`; the search-first Phase 2A that briefly landed
was removed in `v0.2.0`) that collects only what a service cannot exist
without: a name and an image, in the shared `servicefieldsstep`. Enter
writes a minimal two-line fragment via `utils.AddServiceFragment` and opens
straight into the inline editor above. No wizard with a field for ports,
volumes, environment: two fields, then the same YAML a hand-written service
would have. `AddServiceFragment` mirrors `ApplyServiceFragment` (same
fragment shape, same validation, same atomic write) but refuses a name
that already exists rather than one that is missing, and inserts at the
end of the `services:` mapping — the position a reader expects a new
entry in.

Nothing is written unless the fragment parses, keeps its name, and the whole
resulting document still loads as compose — validated by writing a candidate
next to the compose file and running the loader over it. A rejected inline
save keeps the editor open with the error on the status line; the file is
untouched. A rejected `$EDITOR` edit reports the error and returns to a normal
TUI with the file untouched; pressing the key again is the retry. Do not add
a fix-it-before-you-can-leave loop: being unable to leave is worse than losing
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

### Editing group membership

A group's membership can be changed after creation, because creation is not
the moment the user knows the final set. `e` on the groups list reopens the
service checklist — the same component the create flow ends on — pre-checked
with the group's current members. Reusing the checklist is the point:
"which services are in this set" is one question with one answer widget,
whether the set is being made or edited. `e` = "edit the selected thing"
matches `e` on the service details panel.

Saving applies the *diff*, not a rewrite. `utils.SetGroupMembers` reconciles
the file so exactly the checked services carry the tag: newly checked services
are tagged, newly unchecked ones untagged. It runs as a single
read-modify-write pass over the node tree rather than composing the create
and delete walks, which would each open and close the file separately and
leave a crash window with a half-applied edit. Unchecking every service is
allowed and removes the group, because an empty group is the same end state
as a deleted one (§3); create still requires at least one, because a group
that begins empty has no members to derive from.

The request/response split is the one every write uses: the modal emits
`cmds.EditGroupRequestMsg` (the group and the members it chose), `AppModel`
supplies the loaded file and runs `cmds.EditGroup`, and a success reloads the
config so the lists re-derive from disk. A component never learns which file
is loaded — see *One resolution, passed down* under *Which compose file*.

### Deleting a service

Before this, the only way to remove a service's whole entry from the
compose file was to hand-edit it and restart cais — `x` (Remove) only ever
ran `docker compose rm -fs`, which stops and removes a *container*, and
never touched the file; `e` (inline edit) can only replace an existing
service's value node, never delete the key. That gap mattered in practice:
a broken service (bad image, bad config) blocks every group's start (see
§3's untagged-service note above, and the same is true of a broken *tagged*
service the moment its group is picked), and there was no in-app recovery
short of leaving the TUI to edit YAML by hand.

`d` on the services list opens a confirm modal — the same key, same
"delete the highlighted thing" meaning as `d` on the groups list, widened
rather than given a second binding. Confirming runs `utils.DeleteService`,
which removes the service's key from the `services:` mapping and writes
through the same validate-then-atomic-write path every other compose
writer in `src/utils/ServiceFragment.go` uses.

**The unsupported case is a service another one still needs.** Deleting a
service another one names in `depends_on:` would leave that reference
dangling, so the delete is refused rather than silently breaking the file
— the same call rename already makes (see *Editing services*), applied to
the more destructive edit. This cannot be left to the general
validate-by-reload check alone: compose-go's own consistency check
(`loader.checkConsistency`) only walks `project.Services`, the
profile-enabled set for whatever profile the load asked for, and
`ReadConfigFile` asks for none — so a service carrying a `profiles:` tag
loads into `project.DisabledServices` instead and is never visited by that
check. A `depends_on:` between two members of the *same group* is exactly
this shape (both tagged, neither in `project.Services` without
`--profile`), which is the common case for cais's own data model, not an
edge case. `utils.ensureNoDependents` checks every service regardless of
profile — merging `Services` and `DisabledServices` the same way
`AppModel.configSyncCmds` does to build the lists cais shows — before the
node is ever removed, so the general reload check still runs afterward for
everything else a removal could break, but is not relied on for this.

Deleting the only (or last) selected service also exercised a latent gap
in reload handling: `configSyncCmds` used to skip sending
`cmds.SetSelectedService`/`SetSelectedGroup` entirely when the list it was
re-deriving came back empty, on the theory that there was nothing to
select. That left whichever component only updates on those messages —
`detailspanel`, `groupdetailspanel` — holding the *previous* selection
forever, rendering a service or group that no longer existed. Both messages
are now sent unconditionally, with an empty value standing for "nothing is
selected"; `detailspanel.Update` treats the zero `ServiceConfig` (empty
`Name`) as "clear" rather than adopting it as a real service, which is what
`View`'s existing `m.service == nil` check already renders as the "Select a
service" empty card. `groupdetailspanel` needed no equivalent case: its
empty state already keys off `knownGroups() == 0` rather than
`m.selectedGroup`, so an empty `selectedGroup` was already handled, just
never previously sent.

### The Files page

The Files page answers "which file am I acting on, and what is in it?" in
full. It is a single panel — not the two-pane list-and-details split —
showing the active compose file's path on the title row and its raw contents
in a read-only, scrollable viewport. `E` opens the file in `$EDITOR`, reusing
the same handover as the service-details panel (see *Editing services*).

Two things are deliberate. First, the contents are the *raw bytes*, comments
and blank lines included — not the parse tree the groups and services pages
derive from — because the page exists to show the user their own file as they
would see it in an editor, which is also the file `E` is about to open. A
hand-rolled, line-oriented YAML highlighter (`src/highlight`) colors keys,
quoted strings and comments from the active theme as a *display layer over
those bytes* — it changes no byte, so the view still matches the file `E`
opens and the raw text a scroll reveals. It is hand-rolled rather than a
lexer library: the app has one file to color, and Chroma is a heavy
dependency against the minimal-deps stance. It tracks block scalars so a
`command: |` body is treated as literal text rather than structure — a line
like `echo image: x` inside one is not colored as a key, and a `#` inside it
is not a comment — and it is best-effort by design, degrading to plain text
on anything it does not recognize.

Second, the panel is **always focused**: it is the only component on its
page, so there is nothing for Tab to move to, and tracking focus would let
Tab strand it on a component id that does not exist here.

The viewport is fed by a read, not held in app state. `AppModel` issues
`cmds.GetComposeFileContents` when the Files page becomes active and after
every write through the app, so the view re-syncs from disk instead of going
stale. The file's path rides on that message (`ComposeFileContentsMsg.Name`)
rather than only on `SetComposeFileMsg`: a broadcast reaches only the active
page's components, so a Files page that was inactive at load time never sees
the footer-oriented message, but it always sees the read it triggered.

`b` browses the other compose files in the active file's directory. It opens
a picker modal (`ComposeFilePickerModal`) listing every `*.yaml`/`*.yml`
there — not just the four auto-detected names, so it is a way to *choose*,
like `--file`, not a resolution order — with the loaded file marked and the
cursor starting on it. The directory scan runs as a command (the picker
opens off `ComposeFileListMsg`) so `AppModel` does no IO inline. Enter
switches: the modal closes with `SwitchComposeFileMsg` as its follow-up, and
`AppModel` handles that by pointing the source at the chosen path and
reloading with `GetConfig` — exactly what passing `--file` at startup does.
Every downstream consumer (the docker calls, the YAML writers, the footer,
the groups and services lists) already flows from the resolved file, so they
follow without further work; the contents read above repopulates the
viewport on the same chain.

### The Backups page

The Backups page answers "what did this file used to be, and can I have it
back?" in full. It is a single panel with a list on the left and a live preview
on the right, listing every stored copy of the compose file and the `.env`
(`.env` only when one sits next to the loaded compose file) newest first.

A backup is a copy in `.cais/backups/`, with a name of
`UTC-timestamp.sha8-of-content.bak`. `ListBackups` reads both source slugs and
merges them into one list sorted by the embedded timestamp, descending. A source
that has never been written returns an empty list, not an error, so a file with no
history yet just shows the empty state. The preview reads the chosen `.bak` from
disk on selection; compose copies are shown syntax-highlighted through the same
`src/highlight` layer the Files page uses, and `.env` copies are shown raw, so
the preview is the exact bytes a restore would put back.

Restore goes through a confirm modal. `enter` or `r` on a row emits
`RequestRestoreBackupMsg`; `AppModel` opens a `ConfirmModal` whose follow-up is
`RestoreBackup`, which writes the chosen `.bak` back over the live file through
`utils.ReplaceFileAtomically`. Because the write is atomic, the live file is
snapshotted into the store before it is replaced, so a restore is itself
undoable: the copy you restored from still sits in the store, and a later restore
of the post-restore snapshot brings the file you had back. A `.env` restore makes
this explicit in the confirm text, because it brings the secrets back too. A
successful restore reloads the config (so the page and any `.env`-derived
interpolation reflect the restored file) and re-lists the backups so the new
post-restore snapshot appears; a failed one reports a foreground error.

The page reuses the request/response split every other surface uses: the panel
emits an intent (`cmds.GetBackups`, `RequestRestoreBackupMsg`), `AppModel`
supplies the resolved paths and runs the command, and a success message drives a
reload. The panel never learns which file is loaded, for the same reason the
editor handoff does not (see *One resolution, passed down* under *Which compose
file*).

The `.env` editor used to be its own tab. It is now a modal opened by `v`
(`Globals.EditEnv`): `envmodal.New` holds the same variable table, key editor and
raw file editor the page had, plus the reveal/copy/delete keys, and `o` opens the
same inline editor the Files page's `E` does. Moving it off the tab row frees the
fourth tab for Backups, and an env editor is something you open to act on, not a
screen you navigate between, so a modal fits it better than a page slot.

### Home layout

Home is the launchpad. Its body is a two-pane layout:

- **Groups list** — the selectable list of derived groups (Compose profiles)
  with a status header. The empty state is rendered as normal panel text, not
  an inverted box.
- **Group Details** — the right panel. When no groups exist *and the compose
  file has services*, it shows a service overview: a count header and the
  same member-services table a selected group uses, over every loaded
  service, so a user with an ungrouped file can see what they have before
  creating the first group rather than only an explanation of what a group
  is (`docs/plans/service-aware-empty-state.md`). When the compose file has
  no services at all, it falls back to the original onboarding card. When
  groups exist but none is selected it prompts the user to pick one; when a
  group is selected it shows a header card with a status pill, a
  running/stopped/services summary, the member-services table (status dot,
  NAME, IMAGE, STATE, HEALTH, UPTIME, PORTS), and a pinned footer (see *The
  panel footer* below).

The large ASCII logo is no longer rendered here; the About modal carries the
brand mark now, as the figlet banner (`banner.Banner`) - see *Where
keybindings live* above.

### Services layout

Services page is the counterpart to Home for single-service operations. Its
body is a two-pane layout:

- **Services list** — the selectable list of compose services with status and
  memory summary per row.
- **Service Details** — the right panel. Its layout mirrors the polish of the
  Group Details panel:
  - **Empty state:** a *Select a service* card prompting the user to pick from
    the list, using the same recessed-card visual as the group panel.
  - **Editing:** the inline YAML editor replaces the view entirely, showing
    the service's YAML fragment in a `textarea` with live validation on the
    status line below.
  - **Service selected:** a header card in the group header card's shape —
    name (bold), image (muted), and a status line with a coloured dot (●),
    state label, health status (coloured by state) and uptime, separated by
    ` · ` — closing on a thin rule. Every line starts on the body's left edge;
    the image used to be parenthesised and indented by one space, which was the
    only line in either panel that did not.
    
    Beneath the header is a compact two-column table (PROPERTY | VALUE)
    showing the service's compose configuration. Properties are shown in dim
    text, values in primary text, truncated to their column as the member
    table's cells are. Multi-value properties (e.g. ports) indent continuation
    rows below the first value. The table rows shown depend on what the service
    defines, and include: ports, **web** (a real hyperlink to the service's
    resolved URL, directly under ports - see below), container name, restart
    policy, connected networks, volumes (bind mounts by container path,
    named volumes by volume name), healthcheck command (trimmed for
    brevity), depends_on, pull policy, PUID/PGID (common in self-hosted
    *arr stacks), memory limits (in
    human-readable form via `docker/go-units`), and label count.
    
    When the service has a running container with stats data, a live runtime
    stats table (METRIC | VALUE) joins it, showing memory usage + percentage,
    CPU usage, network I/O, disk I/O, PIDs count, and uptime. **The two tables
    sit side by side** once the panel body is 72 columns or wider, and stack
    (separated by a blank row and a rule) below that. Stacked at every width,
    they spent a fifth of the panel's columns and twice its rows: a value
    column seventy blanks wide, under a table that had run off the bottom of
    what the eye takes in at once. Both are the same function
    (`renderPropTable`) differing only in heading and rows; the group panel
    keeps one full-width table because it *has* one table.
    
    While a docker action is pending, a spinner with the action description is
    pinned at the bottom, matching the group panel exactly (see *The panel
    footer* below). Idle, the panel has no footer and its tables run to the
    bottom of the body.

  The service details panel deliberately omits the PUID/PGID row when neither
  is set (since they are optional env-var-derived fields specific to certain
  stacks). The information density was curated for the self-host enthusiast
  audience: ports tell the user how to reach the service, restart policy
  tells them whether it comes back after a crash, volumes tell them what data
  is persisted, depends_on reveals startup ordering, and the resource limits
  help diagnose OOM kills — all common concerns when running a home server.

### A service's URL is a guess, shown with its reasoning

`utils.ResolveURL` is one pure function: a `cais.url` label (used
verbatim, and the only way to say "no, there isn't one" via an empty value;
the rename-era `stitcher.url` is still honored as a fallback) beats a
published port's `app_protocol` beats the tiny fixed set of ports
that conventionally mean https ({443, 8443, 9443}) beats plain http. Among
several published TCP ports, the tie-break promotes a recognised web port
(Jellyfin's 8096, Sonarr's 8989, and so on - a short, deliberately
non-growing table, not a service catalog) over an unrecognised one, in file
order; equally recognised or unrecognised candidates fall back to file
order alone, because the user's own ordering is more meaningful than the
port number. `network_mode: host` ignores `ports:`'s host-side mapping
entirely and is reachable on the container's own port instead. The host
part is resolved once at startup - `config.URLHost` first, then
`SSH_CONNECTION`'s server field (the address this SSH client *measurably*
used, not a guess), then `localhost` - since it cannot change during a run.

**The one sharp edge is rendering it.** `chrome.Hyperlink` wraps
already-truncated-and-padded text in an OSC 8 escape sequence; building the
sequence before truncating would hand `chrome.Truncate`
(`runewidth.Truncate`, not ANSI-aware) a string it could cut through the
middle of, corrupting the rest of the screen. `chrome.Truncate` also grew
an `ansi.StringWidth` fast path so a string that already fits its column -
which a properly pre-sized hyperlink always does - returns unchanged rather
than being re-truncated by the generic row renderer, which stays exactly as
ignorant of hyperlinks as every other row it draws.

### A healthcheck template is a correctness claim, so the catalog stays small

`h` on the Services details panel opens `HealthcheckPickerModal`, listing
`utils.TemplatesFor(image)` — templates whose `Matches` substrings hit the
service's image first, the generic HTTP fallback always last and never
filtered out. Each row is table data (`utils.HealthcheckCatalog`), not
inferred: Postgres, MariaDB, Redis and nginx each name a probe tool the
official image ships, so the catalog never guesses at a container's contents.
The catalog is deliberately not a community directory — a wrong row produces
a container stuck `unhealthy` forever, so small stays safer than complete
(`docs/plans/healthcheck-insertion.md`, "The catalog stays at the 4+1 rows
above").

**One modal, and the port field's visibility is derived, not a focus state.**
The generic template is the only row with a parameter — the container-internal
port, prefilled from the service's first published port's target (else `80`)
— and its input row appears under the list only while that row is
highlighted. No tab, no second modal, no explicit focus toggle: the same rule
covers every key, because "is the field visible" already answers "should this
keystroke type into it." Cursor movement calls `list.CursorUp`/`CursorDown`
directly rather than forwarding keys to `list.Update`, specifically so
bubbles' default list keymap (`h`/`l`/`b`/`u`/`f`/…) never intercepts a
keystroke the port field needs as ordinary typing — the picker's own `h` for
"healthcheck" would otherwise collide with the list's paging binding of the
same letter.

`utils.ApplyHealthcheck` inserts or replaces the `healthcheck:` mapping
directly under the service's value node — the same read-modify-write shape
`SetGroupMembers` uses, not a whole-service fragment, since a healthcheck is
one mapping under the service, not the service itself. It replaces an
existing check rather than producing a duplicate key (two `healthcheck:` keys
in one service is a YAML error), and every template omits `start_interval`:
this app's compose-go parser accepts it, but a user's own `docker compose` CLI
older than 2.20.2 does not, which would make the app's own validation lie
about a file the user's tooling then rejects.

**The apply gap gets a hint, not an action.** `restart` reuses a container's
existing config; only `up -d` re-reads compose. `detailspanel` already knows
whether its service is running (no round trip through `AppModel` needed), so
when `AddHealthcheckMsg` succeeds against a running service it sets the same
footer slot the pending-action spinner uses: `running: press s
to apply - restart won't re-read the compose file`. Auto-running `up -d` on
save was rejected — the editor this hint sits beside edits *any* config field,
and recreating a running container unprompted is destructive; a hint is
reversible, an action is not.

### Boot persistence is a one-key cycle of the compose `restart:` field

`B` on the Services details panel advances the selected service's `restart:`
policy through Docker Compose's four valid values - unset, `on-failure`,
`unless-stopped`, `always` - and back to unset, one step per press. No modal:
unlike `Healthcheck`, which picks from a catalog of image-specific probes,
every value here is already known and there is nothing to fill in, so a
picker would be a screen in the way of a single decision.

This is deliberately the compose-native mechanism and nothing more. Docker
restarts a container on its own once the daemon comes back - after a reboot,
included - when that container carries `unless-stopped` or `always` and the
daemon itself is enabled to start at boot (`systemctl enable docker`, or
Docker Desktop's own login-start setting); cais has no opinion about the
daemon or the host's init system, only about what the compose file says.
A systemd unit that runs `docker compose up -d` at boot was considered and
rejected for now: it would be a second artifact cais owns outside
compose/.env, needs root to install, and only exists on Linux - the opposite
of "nothing extra to host."

`utils.SetRestartPolicy` writes the field with the same read-modify-write
shape `ApplyHealthcheck` uses - find the service's value node, set or remove
one key, validate the whole candidate, then `ReplaceFileAtomically`. Landing
back on the unset state removes the `restart:` key entirely rather than
writing `restart: ""`, which compose does not accept; a hand-edited
`restart: no` is treated as the same unset state so cycling from it does not
get stuck one step behind the rest of the cycle.

`B` reuses the same apply-gap hint `Healthcheck` sets: `restart:` is a
container-level attribute, so a running container only picks up a new policy
on the next `up -d`, not on `restart`. Rather than teach the panel a second
hint, `CycleRestartPolicyMsg` sets `applyHint` through the identical
condition (`AddHealthcheckMsg` and `CycleRestartPolicyMsg` both handled in
`detailspanel.Update`) - one message shape, one hint, two write paths that
share the same caveat.

### The panel footer

Both details panels reserve their body's last line for a footer, laid out by
`chrome.PanelBodyWithFooter` in `src/components/chrome/PanelFrame.go`. Things
that go there: the pending-action spinner, in either panel; and the service
panel's `copied http://…` confirmation after `y` (`docs/plans/service-urls.md`
D6) - the pending-action spinner still wins when both would apply, since a
running action is the more urgent thing to be looking at. Idle with nothing
to report, a panel has no footer at all.

**It is pinned by one layout, not by each panel's arithmetic.**
`PanelBodyWithFooter` takes a panel's content and its footer and pads between
them, so the footer lands on the body's last line whatever the content above it
did. The group panel used to pin it by stretching its member table to fill the
gap and the service panel not at all — it simply followed the last stats row,
which on a tall terminal left it floating mid-panel above a dozen blank rows.
The content is clipped *before* the footer is attached, so a panel whose content
outgrows its box loses its last rows rather than its footer: joining first and
clipping the result takes the bottom off, which is the one row that has to
survive. A running action with no visible spinner is the failure that matters
here, and `TestDetailsPanelsPinPendingActionToBottom` pins both panels to the
same line.

An empty footer costs no rows. `lipgloss.Height("")` is 1, so passing `""`
would otherwise reserve a blank line at the foot of every idle panel.

**There is no action button row.** Both panels used to pin one above the
footer — `s Start`, `t Stop`, `r Restart`, `p Pull`, `x Remove`, `l Logs`, each
a filled chip on `BackgroundRecessed`, the destructive one inked in
`StatusError`, the row dimming as a whole whenever `keys.Active` stopped
offering its keys. It was removed rather than kept for a reason worth recording:
a padded, filled chip is the visual vocabulary of a *clickable* control, and
this app does not handle mouse input. The row's own degradation order argued in
those terms — remove was shed first because "a cramped click target is the last
thing a destructive action should have" — which is a rationale for a feature
that does not exist yet. Mouse support is deferred to a later version, and the
affordance went with it.

Nothing became unreachable. Every key the row showed is on the footer bar,
offered by the same `keys.Active` decision the row was consulting through
`keys.Live` (now gone, as the row was its only caller). What is lost is the
panel's own *local* statement of which verbs act on it — the footer bar is
page-scoped. If that turns out to matter before mouse support lands, the answer
is a plain key-hint line in the footer slot, not the chips back: the hint line
reads as text, which is what a keyboard-only affordance should look like.

The row's *shedding* logic outlived it, and is now the rule three surfaces
follow — see below.

### Narrow terminals: shed whole things

`lipgloss` pads to `Width` but does not truncate. A fixed set of controls
squeezed into a panel narrower than their own labels therefore wraps on the
cell rather than giving anything up, and the result is never "a bit tight" —
it is a heading printed over the next heading, or a row of hints spilling onto
a second line and eating the body underneath. Three surfaces hit this, and all
three now answer it the same way: **drop whole units, in a declared priority
order, until what is left fits.**

| Surface | Order lives in | Never dropped |
| --- | --- | --- |
| Footer bar hints | `keys.Priority` (`src/keys/Keys.go`) | `? help`, `q quit` |
| Group member table columns | `dropOrder` (`groupdetailspanel/View.go`) | the status dot, `NAME` |
| Image reference parts | `ShortImage`'s ladder (`chrome/Image.go`) | the image name |
| *(removed)* action row buttons | the row's own `drop` field | — |

Four rules generalise out of them. `ShortImage` is the first surface where
the shedding happens *inside* a unit rather than between units — the unit is
the image reference, the parts are registry, namespace and tag. The rule
survives the move down a level: a part is whole or absent, never a fragment
(`lscr.io/linuxse…` was the defect), which is why a plain `Truncate` is
rung 4, the last resort, and not the mechanism.

**The drop order is not the display order.** The order to read in is not the
order to give up: the footer shows `1-3 page` beside `q quit` and sheds it
first, because the nav bar prints the digits already; the table shows `IMAGE`
second and drops it fifth-from-last, because a row is identified by its name,
not its tag. Writing the two orders separately is what lets each be argued on
its own terms, which is why both are tables with reasons attached rather than
a sort at render time.

**Something has to survive that leads back to what was shed.** This is what
makes shedding safe rather than merely lossy. On the footer that is `? help`,
which opens the overlay listing every binding — so a shed hint is hidden, not
lost. In the table it is `NAME`, because a state with nothing to attach it to
says nothing at all. A surface with no such anchor should not shed; it should
scroll or paginate.

**A unit is whole or absent, never a fragment.** `NAMEIMAGSTATHEALT…` and a
hint cut mid-word are the same defect as the wrap they replaced. Cells
truncate one column short of their width so two values keep a gap between
them, and every one of these surfaces has a test that walks descending widths
asserting nothing renders as a fragment and nothing shed ever comes back.

**`MaxHeight` is the backstop, not the fix.** Every one of these surfaces
clips to its own height as a last resort, for the widths below which even the
never-dropped units do not fit. Clipping keeps the layout intact but says
nothing about what was lost, so it is what happens *after* the priority order
has run out — reaching for it first is what let the action row wrap to
thirty-one rows inside a panel that looked fine from the outside, absorbed by
eating the member table.

`TestNarrowPanelsStayInsideTheirBox`, `TestFooterNeverWraps` and
`TestMemberTableHeadingsNeverCollide` are the standing guards.

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

`appstyles.Themes` is the registry (14 themes: 3 Cais themes
cais-dark, cais-dusk and cais-day, plus 11 community schemes), built
by `appstyles.newTheme` from
a handful of base colors - `Accent`, the text/panel/modal bases, `Danger`,
the four status colors - with everything else derived by `Lighten`/`Darken`.
A dark theme raises a tier's attention by lightening it, a light theme by
darkening it; see the constructor's doc comment for why both directions use
the same deltas. Adding a theme is choosing those base colors, not hand-
tuning thirty derived ones.

**The asymmetry that drives every imported palette:** `Lighten` is additive
(+10/+20/+31 per channel at the standard deltas) and `Darken` is
multiplicative (×0.96/×0.92/×0.88). For a dark theme this is a fixed climb
independent of the base; for a light theme the steps shrink as the base
approaches white. The consequence for imported colour schemes: **set `Panel`
to that scheme's deepest background tier** (crust / bg_dim / bg0_hard /
sumiInk0 / bg_dark), so the +8% tier (`BackgroundPanel`) lands back on the
scheme's signature background. `Modal` must clear `BackgroundElevated` by
≥14 per channel or the modal disappears into the focused panel.

Two fields are the one deliberate exception to "derived from base colors":
`InkOnLight`/`InkOnDark` do not vary with a theme's `Dark` flag, because a
status pill's own fill (`StatusRunning` green, `StatusStarting` amber,
`StatusError` red) does not vary with the app's theme either - the text that
reads legibly on a green pill has to stay dark whichever theme is active, not
follow `TextPrimary`, which flips. `groupdetailspanel/View.go`'s `statusPill` used
to reach for `PanelBg`/`TextPrimary` as stand-ins for "a dark color" and "a
light color"; that only ever worked because the one theme that existed was
dark, and the first light theme is exactly what exposed it.

With the expanded registry, hard-coding which ink to use on a given fill is
no longer survivable — the same call site draws on a `#BC3FBC` magenta in
one theme and a `#A7C080` sage in another. The `appstyles.InkOn(fill)` helper
picks whichever of the two fixed inks has better contrast on the fill at
hand, and `Contrast_test.go` verifies the result clears 4.2:1 on every status
pill, the accent chip, and the error banner for every registered theme.

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

### Theme picker modal

`T` (shift+t) opens a modal listing every registered theme, sorted by name,
with the active one marked and the cursor starting on it. Cursor movement
previews live: `ThemePickerModalModel.Update` detects an index change after
the list update and calls `appstyles.SetTheme`, so the entire UI behind the
modal repaints on each keystroke. The original theme is captured at
construction time, so `Esc` always restores what the user started with,
even after several preview steps. `Enter` applies and persists: the choice
is written to `~/.config/cais/config.yaml` via
`config.SaveConfig`, and the saved theme is loaded in `main.go` before
the program starts — a missing or malformed config silently yields the
default. Persistence errors surface in the banner rather than blocking the
modal close, because the theme the user saw is already active.

The config struct (`src/config/config.go`) has one field (`Theme`) today
but is designed to absorb the other post-alpha preferences (default file,
keybinding overrides) without changing existing callers: add a field, tag
it, and `LoadConfig`/`SaveConfig` round-trip it automatically.

Adding a theme is choosing the handful of base colors in `themeParams`
(accent, text, panel, modal, danger, the four status colors); `newTheme`
derives every other field. The 14 registered themes are cais-dark,
cais-dusk and cais-day plus 11 community schemes (Catppuccin Mocha,
Catppuccin Latte, Gruvbox Dark, Gruvbox Light, Tokyo Night, Nord, Dracula,
Solarized Dark, Everforest Dark, Rosé Pine, Monokai Pro). The three Cais themes share
one set of status and danger colors on purpose: container state is a
vocabulary the user shouldn't have to re-learn between them. An imported
scheme brings its own, because a Cais teal dropped into Gruvbox would read
as the one thing on screen that isn't Gruvbox; what stays constant there is
the *mapping* (green runs, amber starts, red errs), not the hex.

### Saying which build this is

`constants.Version()` is the single reader of a version that may or may not
have been stamped, so no caller has to care whether it was. `make build` and
the GoReleaser build both pass `-ldflags -X …constants.version=`, and that
value wins when it is there.

When it is not, the build info answers, and which half of it answers separates
the two remaining cases without a heuristic. A binary built from a checkout has
a `vcs.revision` — the short commit is its version, and it is the thing a bug
report actually needs. A binary from `go install …@v0.1.0` has no VCS
information at all, because it was built from a module download, so
`Main.Version` is both the only answer and the right one.

The toolchain's synthesized `v0.0.0-<date>-<hash>` pseudo-version is
deliberately ignored: it says exactly what the commit says, three times longer,
and looks like a release nobody ever made.

The nav bar renders it dimmed, left of the wordmark, and drops it when the row
gets tight — the same bargain the footer makes with the compose file name, and
for the same reason: the tabs are what the nav is for.

### Docker's absence is a diagnosis, not an error

Every docker call in the app is `exec.Command("docker", …)` followed by
`CombinedOutput()`, which used to mean five distinct, differently-fixed
problems — docker missing, the Compose plugin missing, the daemon stopped,
the socket unreadable, or a healthy machine — all arrived as one line of
`exec.ExitError` noise. `utils.DockerPreflight` classifies which of the five
it is; `dockerstatusmodal` says so and prints the exact command that fixes
it, copyable, never run.

Classification comes from *which probe failed*, not from parsing what it
printed. Docker rewords its errors between majors — the daemon-down message
changed completely between 28 and 29 — so a classifier built on matching the
whole string would degrade silently the next time docker changes its
wording. Exit codes and probe order do not reword; the one exception is the
split between "daemon down" and "permission denied", which needs a single
`strings.Contains(out, "permission denied")` with a documented fallback to
"daemon down" when nothing recognisable is found.

The app never installs, starts, or configures anything on the user's
machine, even behind a confirmation prompt. There is no single correct
install across apt/dnf/pacman/Docker Desktop/colima/rootless, the one-liner
installers are not what their own vendors recommend for anything but a
throwaway box, and a TUI that asks for root to fix a problem it just
diagnosed looks exactly like the thing a user should never type their
password into. The diagnosis plus an exact command is most of the value at
none of the blast radius, and the decision to change the machine stays with
the user.

The probe runs once at startup and again on every docker error: `docker
compose ps` failing is re-probed before it is reported, and a docker action's
own failure kicks off the same re-probe alongside the raw error it puts up —
if the re-probe comes back with a diagnosable state, it replaces that raw
error with the diagnosis rather than leaving `exit status 1` as the last
word. A healthy re-probe (the daemon is up but wedged) leaves the raw
message standing, because the app has no diagnosis to offer for a state it
cannot reproduce. This is what makes the feature the app's permanent answer
to "docker broke" rather than a first-run nicety: daemons stop, sockets get
replaced by an upgrade, laptops sleep, and the diagnosis is available every
time, not just once at boot.

See `docs/plans/docker-preflight.md` for the full research (exact strings,
exit codes, timings) and the platform remediation table.

## 6. Package layout

`src/model` is the app: `AppModel`, its `Init`/`Update`/`View`, and the
message-routing that owns every screen. `src/components` is the leaf
models it composes — one folder per model (`serviceslist`, `detailspanel`,
`groupnamemodal`, …), each holding `Model.go`, `Update.go` and `View.go`
split out once the model earns it (roughly 150 lines, or `View` growing its
own render helpers — a 60-line model stays one file). The constructor is
always `New`; the exported type is always `Model`, so callers read as
`serviceslist.New(...)` and assert on `serviceslist.Model`.

`src/components/chrome` is the one shared package: rendering and layout
that more than one model needs (`PanelFrame`, panel body layout, key-hint
rendering, the spinner, `HealthColor`/`Truncate`). A helper earns its way
into `chrome` by having a second caller, not by convenience — a helper used
by exactly one model stays unexported inside that model's package. This is
enforced by the compiler, not by convention: an unexported helper simply
cannot be reached from outside its package, so a second caller forces the
decision explicitly rather than letting it drift.

No model package imports another's internals; the only inter-model
reference in the tree is `groupnamemodal` constructing
`servicechecklistmodal.New` at the handoff point in the create-group flow,
both through exported API. Only `src/model` imports the leaf packages —
nothing downstream imports `src/components`, so `chrome` cannot become part
of an import cycle no matter who ends up depending on it.

## 7. Decision checklist for new features

Before adding a feature, answer these:

1. **Group or service?** See §1. If neither, it probably belongs on a new
   page (e.g. monitoring, settings).
2. **Can it be expressed as `profiles:` tag manipulation?** See §3. If yes,
   it fits the home page's mental model. If no, it may need a different
   mechanism.
3. **Does it require editing existing services?** Inline (`textarea`) editing
   is currently a roadmap item; the `$EDITOR` path (`e` for one service, `E`
   for the whole file) already works. The first place to add inline editing is
   the Services page, not home — home is for groups.
4. **Does it conflict with the Services page's role?** If a feature would
   duplicate Services-page functionality, prefer to extend that page unless
   there's a clear group-level reason.

## 8. Related documents

- [Roadmap](ROADMAP.md) — the ordered plan, the decisions already taken with
  the owner, and the post-alpha list. Live.
- [Current TODO](../TODO.md) — the live worklist and recent completed work.
- [Contributing](../CONTRIBUTING.md) — the build/test loop, how to test a TUI,
  and how a release is cut.
