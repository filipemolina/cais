# Backups Page Rework — Implementation Plan

## Status

Phases 0-5 have landed; the plan is complete. The codebase-consistency work
(`docs/plans/codebase-consistency.md`) ran to completion on top of Phase 3;
the working tree is clean and the suite is green.

| Phase | Commit | |
| --- | --- | --- |
| 0 — the keymap rule | `88c00b2`, `ef22549` | done |
| 1 — split into two panels | `0ada178` | done |
| 2 — scroll the list | `8926ae1`, `074c0ad`, `c3f5c2b` | done |
| 3 — focus, and the bubbles-list conversion | `4cb09ec` | done |
| 4 — the diff engine | `d14ceda`, `33fba9d`, `687df98` | done |
| 5 — render the diff | `bd286c0`, `beab530` | done |

### Picking this up cold

Read *Decisions*, then Phase 4 and Phase 5, then *Traps found the hard way* —
the traps section is the part that will cost you time if you skip it.

The Backups page is `AppModel.pages["Backups"]`: two sibling components,
`src/components/backupslist` (a bubbles list over `utils.BackupEntry`, owns
the cursor) and `src/components/backuppreviewpanel` (a viewport over the
selected copy's bytes, owns the read). They never reference each other; the
cursor reaches the preview as `cmds.SetSelectedBackupMsg`. `AppModel` owns
`backupsFocus` and is the single handler for `tab`.

Phase 4 adds the diff and touches neither the list nor focus. It is scoped to
`backuppreviewpanel` plus a new package for the diff itself, so the seam is
clean.

**The consistency-audit gate below is settled — Phase 4 may start.** The
codebase-wide consistency audit commissioned right after Phase 3 landed ran,
and its plan (`docs/plans/codebase-consistency.md`) has been acted on in full:
every task landed, with its deliberate deferrals recorded there. Nothing Phase 4
builds on is expected to move. One deliberate near-neighbour worth knowing
about: the Backups version list keeps its own row delegate rather than the
shared `chrome.ListRow` (its rows carry a focus tier), and Phase 4's list-row
marker is drawn by that same delegate — this plan does not fold it either.

Two corrections landed on top of Phase 2 that this plan did not call for,
both from review rather than from the plan:

- `074c0ad` gave the rows the groups/services row language — a state bar
  carrying colour alone, `Padding(1)`, bold on the cursor row. The bar was
  previously drawn in the row's own background colour, which made it
  invisible rather than grey, so this fixed a bug as well as a mismatch.
- `c3f5c2b` put the live file's own name on each row. `BackupEntry.Source`
  is a routing key a restore resolves back to a live path, so every compose
  filename collapses to `"compose"` and `compose.yml` cannot be told from
  `compose.yaml`. `BackupEntry` gained a `File` field for the basename;
  `Source` kept its job. The sha left the list for the preview's title, and
  the column header went with it.

All four defects in *Problem* below are now fixed: 2 and 4 in Phase 2, and 1
(no keyboard scrolling of the preview) and 3 (the footer advertising an inert
`esc`) in Phase 3.

The version list was then moved onto the bubbles list, in the same commit as
Phase 3, so all three body lists are set up the same way. It had hand-rolled
its own cursor, windowing and paging since Phase 1, on two justifications that
did not survive checking:

- *"the rows are two-line entries, not `list.Item`s"* (Phase 1). The services
  list renders 4-line rows through a custom delegate, and `074c0ad` had already
  made the backups rows match it exactly.
- *the 39ms benchmark* (Phase 2). It measured a `viewport.Model` holding the
  whole rendered list. The bubbles list renders `items[start:end]` - one page -
  which is the same windowing Phase 2 hand-rolled.

Phase 4 landed as three commits, with four things worth recording:

- **`d14ceda` — the engine.** `src/diff` wraps go-udiff into the whole-file
  line sequence. The plan's sketch named `udiff.ToUnifiedDiff` as producing a
  string; in v0.4.1 it returns a structured `UnifiedDiff` (hunks and lines),
  so the engine flattens the structured form instead of parsing a diff string
  back. `Line.Content` loses its trailing newline - the renderer draws one
  line per row, and a newline inside Content would be a blank row. Identical
  inputs come back as all-Equal lines, which is what the list's "current"
  marker is built on. A context count at least the file's own line count
  merges every hunk into one spanning the whole file.
- **`33fba9d` — the live reads.** `GetBackups` now reads each live file once
  per list load and carries `utils.LiveSource` (bytes + the same sha8 the
  `.bak` names use) on `BackupListMsg`; `SnapshotFile` and the reader share
  `utils.ContentSHA8` so the two ends of the comparison cannot drift. The
  preview computes the diff on whichever side arrives, and re-computes on a
  re-list - a write while on the page re-lists through the `GetConfigMsg`
  handler, replacing the live side without moving the cursor. The direction
  is pinned at the panel: `diff.Lines(live, copy)`, so Insert is what
  restoring would add to the live file and Delete what it would remove.
- **`687df98` — the marker.** Rows whose content hash matches their source's
  live hash carry a `(current)` marker beside the filename, one emphasis step
  below the title, never the accent. It is dropped whole when the panel is
  too narrow, and drawn by the version list's own (unfolded) row delegate.
  DESIGN.md records it alongside the page's other behaviour.
- **One interpretation of the plan, for the record:** the "identical to the
  live file" empty state is *computed* in Phase 4 - the panel holds
  `lines []diff.Line`, where nil means unavailable - but it is not *rendered*
  until Phase 5, because an empty state for a diff only makes sense once the
  viewport shows the diff. Phase 5 should render it from `lines` (or the
  panel's sha comparison) rather than re-derive the inputs.

Phase 5 landed as two commits - the derived colors, then the rendering -
with four things worth recording:

- **`bd286c0` — the washes.** The plan called for deriving the colors via
  `lipgloss.Blend1D`, which leaves the blend fraction to choose. 28% toward
  the status color is where the two floors stop fighting: solarized-dark's
  body text clears 4.5 on the wash (4.60, the tightest theme) while the
  tightest wash stays visible against the elevated tier (everforest-dark, 8
  channel units - the tier ladder's own floor). The contrast test pins text
  on the wash, the wash against both tiers it renders on, and the gutter
  glyph's status color on its wash - all per theme.
- **`beab530` — the render.** Equal lines consume the copy's own highlighted
  output in step with the diff (which is also what carries the YAML
  colorizer's block-scalar state across); Insert and Delete lines render
  raw, because a changed line's wash replaces its syntax colors. An Insert
  initially took the styled text like an Equal line and would have put a
  colored key inside a colored row - the exact collision *Decisions*
  forbids; the render tests caught it before it landed. The gutter marker
  takes the full status color on its wash, not the wash color itself -
  DiffRemove on DiffRemove is one color, an invisible marker.
- **The hooks are installed per render.** `StyleLineFunc` and
  `LeftGutterFunc` close over the model, and a value model's Update return
  is a copy - a hook stored on `m.vp` would read through a stale one. The
  viewport is copied in `View` and the hooks installed there, so they always
  see the frame's own lines.
- **The trap measured.** `BenchmarkTheDiffRenders` drives both a cursor move
  (selection, read, recompute) and a page keystroke at a 1000-line ceiling:
  the frame cost is flat from 100 to 4000 lines and within ~0.2ms of the
  pre-diff baseline - the per-line styling runs over the visible window, not
  the file.

The empty state came out of the same commit as the render, read off `lines`
(all-Equal) as the Phase 4 note above anticipated, so it cannot disagree
with what the viewport would have shown. The VHS recordings against the demo
fixture stack caught one thing no unit test had: the `.env` diff's secrets
sit in washes in both directions, plain text per the standing constraint
DESIGN.md now records.

What that bought: filtering (`/`, on the file, the timestamp *and* the sha,
none of which the rows all show), pagination, `h`/`l`/`←`/`→` paging, and one
keymap - `keys.ListKeyMap` - across all three lists, which is the rule Phase 0
exists to enforce. `ensureCursorVisible`, `rowOffset`, `visibleRows`, `pageBy`
and `halfPage` are gone with their tests. Two behaviour changes came with it:
the list pages rather than scrolls a row at a time, matching the other two, and
`ctrl+u`/`ctrl+d` no longer page the list, since `list.KeyMap` has no half-page
binding (the preview keeps them).

Phase 3 landed as written, with three things worth recording:

- **The focus flag went into `apptypes`, not into either panel.** Four
  packages have to agree on it — `AppModel` owns it, `cmds` carries it, `keys`
  reads it for the arrow hint, both panels route by it — so
  `apptypes.BackupsFocus` is the shared vocabulary. `chrome` could not hold it:
  `chrome` imports `keys`, so `keys` cannot import `chrome`.
- **`PanelBgFor` needed `PanelFrameOn` beside it.** The plan called for the
  background helper but `PanelFrame` fetched its own `PanelBg()`, so the lift
  would have reached the body and not the frame around it. Same for the list's
  rows, which take the panel's tier as a parameter now
  (`chrome.ListRowBgOn`) rather than assuming the elevated one — DESIGN.md's
  standing rule for anything drawn inside a panel.
- **The bubbles list needs two pagination passes after `SetItems`.**
  `list.updatePagination` sizes a page against a height it has taken the
  paginator's row out of, but `list.paginationView` measures a bare line until
  it knows there is more than one page - so the first pass after rows arrive
  fits one row too many and the next pass silently disagrees. The panel drew
  four rows a page while focused and three the moment anything touched the
  delegate. `backupslist.setItems` re-runs it. The groups and services lists
  get their second pass by accident, from the `SetDelegate` inside
  `syncActiveIndex`, which only fires when the active row actually moved.
- **Unhighlighted preview text had no foreground.** An .env copy is shown raw
  so a restore's exact bytes are on screen, but "raw" was being taken to mean
  uncoloured too: the text carried no SGR and fell back to the terminal's own
  default foreground, which is pale grey on a light theme's pale panel. This
  predated the phase - it was visible on the merged page as well - and is
  fixed with `backuppreviewpanel.plainText`.
- **The rig's screen decoder was missing `CSI G` and `CSI X`.** The end-to-end
  test drove the renderer into column-absolute deltas that
  `src/model/screen_test_util.go` silently mistracked, reconstructing a
  plausible but wrong screen. The decoder now handles CHA, ECH and CUB, with
  cases in `TestScreenDecoder`. It still cannot place the confirm modal's
  frame, so that one assertion matches the raw stream and says why.

## Problem

The Backups page answers "what did this file used to be, and can I have it
back?" — but it answers it with a wall of undifferentiated YAML. The page's
one verb is `r` (restore), and nothing on screen tells you what restoring
would actually change. You are asked to eyeball a diff.

Investigating that turned up three latent defects on the same page, all of
which have to be fixed before or alongside the feature:

1. **The preview cannot be scrolled with the keyboard.**
   `backuppage/Update.go` routes every key press into `handleKey`, which
   ends in `return m, nil`. Nothing ever reaches the viewport branch below
   the switch. The keymap built by `previewViewportKeyMap` — `pgup`,
   `pgdown`, `ctrl+u`, `ctrl+d` — is dead code. Only the mouse scrolls.

2. **The version list clips instead of scrolling.**
   `View.go:renderList` renders *every* entry, then `PanelBodyWithFooter`
   applies `MaxHeight` (`chrome/PanelFrame.go:69`), which truncates from the
   top with no scroll offset. Each row is two lines, so roughly 14 entries
   fit a 30-row panel. Past that, `j` keeps advancing `selectedIdx` with the
   preview updating while the highlighted row is off-screen and
   unreachable. `G` jumps to an entry that is guaranteed invisible.
   `MaxBackupsPerSource` is 500 across two sources, so this is reached after
   a couple dozen edits.

3. **The footer advertises an inert key.**
   `keys.Active()` returns `Global.Back` for `"Backups"`, but the esc ladder
   (`model/Update.go:490-499`) has no Backups branch — esc only dismisses an
   error banner here. DESIGN.md: "the bar does not advertise inert keys."

4. **`previewViewportKeyMap` contradicts itself.** Its doc comment says
   `k`/`j`/`↑`/`↓` stay with the list; two lines later it binds `Up` to
   `"up","k"` and `Down` to `"down","j"`. Invisible today only because
   nothing reaches the viewport.

## Solution

Rebuild the page as **two real panels** with **Tab-switched focus**, and give
the right-hand panel a **git-style diff** of the selected copy against the
live file.

- Left panel: a scrolling list of stored versions, its own component.
- Right panel: the selected copy rendered as a diff against the live file —
  changed lines tinted red/green, unchanged lines keeping the existing YAML
  highlighter, auto-scrolled so the first change lands mid-screen.
- `tab` / `shift+tab` move focus between them. `esc` gets no new behavior:
  Backups is a page, and `1`-`4` leave it.

Because the two panels become two entries in `pages["Backups"]`, the generic
body layout in `model/View.go:renderBody` gives the gutter, the per-panel
sizing and the draggable split for free.

## Decisions

Settled during design; recorded here so they are not relitigated.

**Diff against the live file, not against the previous backup.** The page's
one action is restore, so the diff that matters is "what does restoring this
do to my file right now?" Diffing against the next-newer backup answers a
question the page gives you no way to act on.

**Diff colors replace syntax colors on changed lines.** Context lines keep
the `src/highlight` YAML colorizer. Green-on-green plus key coloring is too
much signal at once, and a changed line's meaning is "this changed", not
"this is a mapping key".

**`.env` diffs show secret values in plain text, deliberately.** The page's
existing contract is that the preview is the exact bytes a restore would
write. A diff inherits that contract. Any future change to this view must
not introduce masking that would break it. This goes in DESIGN.md as a
standing constraint, not a footnote.

**`go-udiff` rather than a hand-rolled diff.**
`github.com/aymanbagabas/go-udiff` is already in the module graph — required
by `bubbles`, `bubbletea` *and* `lipgloss`, pinned at v0.4.1, already in
`go.sum` and the module cache. It has zero dependencies of its own and is
2,222 lines: a port of the Go standard library's own `internal/diff`.
Promoting it to a direct require adds no module to the build list and no new
maintainer to the trust boundary.

This does not weaken the minimal-deps stance that kept Chroma out of
`src/highlight`. Chroma is a general-purpose highlighter carrying 200+
lexers, embedded style data and a third-party regex engine, to color one
language, whose styles would then need remapping onto 15 themes. The two
cases differ on every axis that matters. A hand-rolled line-oriented YAML
colorizer that degrades to plain text is honest and bounded; a hand-rolled
Myers diff has edge cases (long common runs, whitespace-only churn, the
O(ND) bailout) that surface months later on someone's real compose file.

**`r` is the only restore key.** `Backup.Restore` currently binds `enter`
and `r`. `enter` is too easy to hit by reflex while navigating for an action
that overwrites a live file. It is dropped.

**`r` stays live regardless of focus.** DESIGN.md's own focus-removal
rationale says a verb "was never about which panel, only which row is
selected" — and focus does not move the selection.

**Intra-line highlighting is deferred.** Whole-line tinting first. See
*Deferred* below for the seam that keeps it from becoming a rewrite.

## Phase 0 — The keymap rule (app-wide, independent)

Forwarding a message to a child component is not the defect. Forwarding into
a **library default keymap** is. `keys.ListKeyMap()` and
`composeFileViewportKeyMap()` already do this correctly, and
`ListKeyMap`'s doc comment records the exact symptom the rule prevents:
"pressing d both opened the delete confirm and paged the list backwards."

Add to DESIGN.md, under the keybinding conventions:

> A component may forward keys to a child only if that child's keymap was
> explicitly constructed. No component ships a child carrying
> `list.DefaultKeyMap()` or `viewport.DefaultKeyMap()`.

What the defaults silently claim:

- `viewport.DefaultKeyMap()` — `f b u d h l j k space` plus arrows/pgup/pgdn
- `list.DefaultKeyMap()` — `q esc ? / h l b u f d g G tab shift+tab ctrl+k
  ctrl+j ctrl+c` plus arrows

Six sites forward into a default keymap. All are modals or the containers
list, where the surface owns the whole keyboard, so the blast radius is
undeclared keys rather than verb collisions — real, worth fixing, not on
fire:

| Site | Child |
| --- | --- |
| `logsmodal/Update.go:69` | `viewport.DefaultKeyMap()` |
| `themepickermodal/Update.go:38` | `list.DefaultKeyMap()` |
| `containerslist/Update.go:41` | `list.DefaultKeyMap()` |
| `servicechecklistmodal/Update.go:41` | `list.DefaultKeyMap()` |
| `composefilepickermodal/Update.go:29` | `list.DefaultKeyMap()` |
| `healthcheckpickermodal/Update.go:54,55` | `list.DefaultKeyMap()`, though inertly — see below |

Give each an explicitly constructed keymap naming only the keys that surface
actually uses.

`healthcheckpickermodal` is the exception and was mischaracterised in an
earlier draft of this plan. It never forwards a *key* to its list: every key
path returns before line 54, which is reached only by cursor blinks and
window sizes, both of which legitimately need to reach the list and the port
input. Its comment at lines 23-26 already documents avoiding the default-map
collision by hand. It still gets an explicit keymap, as defense in depth and
so the invariant is uniform, but it is not a live bug.

`containerslist` is dead code — nothing outside its own package references
it, left behind by `75679d5 refactor: remove panel focus management`. It gets
a keymap here for uniformity; whether to delete the package is a separate
decision.

The rule is testable, and the test should be source-level rather than
per-package: the keymaps sit on unexported fields, so a per-package test has
to be remembered for every new component, which is the discipline the rule
exists to replace. Walk the component tree and fail on a `list.New` or
`viewport.New` whose result never gets a keymap.

## Phase 1 — Split the page into two components

Replace `pages["Backups"] = []tea.Model{backuppage.New()}` with two entries.
`renderBody` already loops over the slice inserting gutters, so this is
where the two-panel layout, the per-panel sizing and the draggable split
come from.

- `src/components/backupslist` — owns `entries` and `selectedIdx`, renders
  the version rows. Its own component, not a reuse of `groupslist`: the rows
  are two-line source/timestamp/sha entries, not `list.Item`s, and it needs
  no filter.
- `src/components/backuppreviewpanel` — owns the viewport, the read of the
  selected copy, and later its diff rendering. Named for the job it keeps
  across every phase (show the selected copy) rather than for the coloring
  Phase 5 adds, so no rename is needed later.

Selection moves between them the way Home already does it: the list emits
`SetSelectedBackupMsg` on cursor move and AppModel broadcasts it to the
active page's components, so the two panels never hold a reference to each
other. AppModel needs no state of its own for this — restore already carries
the source and name it acts on. The list stops reading `.bak` bytes; the
preview panel does, since it owns the viewport they land in.

Reads are tagged with the path they were issued for, so one that finishes
after the cursor has moved on is discarded rather than painting one copy's
bytes under another copy's header.

No behavior change beyond the layout in this phase. Land it green, with the
existing tests ported, before anything else moves.

One trap for the ported tests: the model-level helper `drive` applies
messages but never runs the commands they produce, and this page is now a
four-message chain (activate → GetBackups → BackupListMsg → the list
publishes → the preview reads). A test that stops after the first message
renders zero-width panels and asserts nothing. `settle` in
`src/model/backup_test.go` runs the chain the way the runtime does.

## Phase 2 — Make the list scroll

Fixes defect 2, and it is the one phase that fixes a bug users hit today. It
is independent of the diff and of focus, and should land on its own.

The list keeps a `rowOffset` and renders only the rows inside the window,
with a cursor-follow that scrolls the least amount that brings the selected
row into view. The window is tracked in whole rows, never half of one: a row
is a timestamp line and a sha line, and scrolling to show half of it is
worse than not scrolling.

**Not a `viewport.Model`,** which is what this plan first called for. The
selected row is styled differently from the rest, so a cursor move restyles
two rows, which means the viewport's content has to be rebuilt on every
keystroke. At the store's ceiling - `MaxBackupsPerSource` across two sources -
that measured **39ms per cursor move**, well past a frame and visibly laggy
on a held key. Rendering only the window costs about 2.5ms for a whole frame
and, more to the point, does not grow with the history:

| entries | frame |
| --- | --- |
| 10 | 1.98ms |
| 100 | 2.45ms |
| 1000 | 2.31ms |

`BenchmarkRenderRowsAtStoreCeiling` keeps that honest.

Worth a table test at the boundaries: first row, last row, exactly-fits,
one-past-fits, and a resize that shrinks the panel under a low cursor.

The rows also adopt the groups/services row language, since they are the
same kind of thing on the same kind of page: a state bar down the left edge
carrying state by colour alone (accent for the cursor, muted grey
otherwise), content set in a wrapper with `Padding(1)`, and the cursor row's
title in bold. Two content lines inside that padding makes a row four lines
tall, the same as a services row.

## Phase 3 — Focus

`AppModel` gains a `backupsFocus` field, valid only while
`activePage == "Backups"`, and handles `tab` / `shift+tab` scoped to that
page — AppModel does not currently match `tab` at all
(`model/Update.go:466-468` is a tombstone comment where the old handling
was), so nothing is being taken from anywhere else. Handling it in one place
rather than in both panels keeps a single owner for the key, per Phase 0.

The new focus is broadcast to both panels, which route their own keys by the
flag:

| Key | List focused | Preview focused |
| --- | --- | --- |
| `↑` `↓` `j` `k` | move cursor | scroll one line |
| `g` `G`, `home` `end` | first/last entry | top/bottom of file |
| `pgup` `pgdn` `ctrl+u` `ctrl+d` | page the list | page the file |
| `r` | restore | restore |
| `tab` `shift+tab` | → preview | → list |

**Paging the list is manual arithmetic on `rowOffset`,** not a viewport
call. Phase 2 deliberately did not give the list a `viewport.Model` (see
above for the measurements), so there is no `HalfPageDown` to delegate to.
It is a few lines against `visibleRows()`, and the list already matches
every key it handles explicitly.

The preview panel does have a real viewport, already carrying
`keys.ReadOnlyViewportKeyMap()`, so its half of the table is a matter of
routing keys to `m.vp.Update` when it holds focus - which is also the point
where the dead code noted in defect 1 finally starts running.

Focus is shown with the documented tier lift: the focused panel sits on
`BackgroundElevated`, the unfocused on `BackgroundPanel`. `chrome.PanelBg()`
returns elevated unconditionally today (focus having been removed
app-wide), so add a `chrome.PanelBgFor(isFocused bool)` used only by these
two panels rather than changing `PanelBg()`'s signature across every
component.

Also in this phase: drop `Global.Back` from `keys.Active()`'s `"Backups"`
case (defect 3 — `esc` does nothing on this page, and DESIGN.md says the bar
does not advertise inert keys), drop `enter` from `Backup.Restore` so `r` is
the only restore key, and add a scroll hint to the footer modelled on
`Files.Scroll`. `keys.Active` will need a focus dimension to say which half
the arrows are driving.

The viewport keymaps already moved into `src/keys/Keys.go` in Phase 0, and
defect 4 went with them, so neither is outstanding.

## Phase 4 — The diff

A new `src/diff` package, pure and UI-free:

```go
func Lines(before, after string) []Line

type Line struct {
    Kind    Kind   // Equal, Insert, Delete
    Content string
    Spans   []Span // intra-line ranges; always empty until Phase 5
}
```

Built on `udiff.Lines` plus `udiff.ToUnifiedDiff` with a large context count
(the panel shows the whole file, not hunks — auto-scroll only makes sense in
a full-file view). `Spans` is present and empty from day one so Phase 5 fills
a field rather than changing a signature.

Wiring: the diff needs the **live file's** contents for the matching source.
Use `entry.Source` (`"compose"` or `".env"`) for that, not `entry.File`:
Source is the routing key AppModel already resolves to a live path, which is
exactly the job here. `File` is for display. The panel deliberately does not
know which files are loaded — AppModel supplies resolved paths, per the
request/response split — so live contents ride in on the message rather than
being read by the panel.

Cheap win from the existing store: the `.bak` filename already carries the
content SHA8, so hashing the live file once per list load gives both an
honest "identical to the live file" empty state and a marker on the list row
for "this is what you have now".

## Phase 5 — Render the diff

Theme colors are **derived, not hand-picked**. Adding `DiffAdd`/`DiffRemove`
across 15 themes would fight `newTheme`, whose whole premise is deriving
thirty colors from about ten. Derive them in `newTheme` from `StatusRunning`
and `Danger` via `lipgloss.Blend1D(n, PanelBg, …)`, so each theme gets a
tint that sits correctly on its own surface. Keep them as separate `Theme`
fields rather than using the status colors directly at the call site: "a
container is running" and "this line was added" are different concepts that
happen to share a hue. Extend `TestWCAGContrastAgainstSurfaces` to cover the
new pairs — it already loops every theme, so it is a few lines, and it will
catch the theme where the tint eats the text.

Rendering uses two viewport hooks made for this: `StyleLineFunc` for the
per-line row tint and `LeftGutterFunc` for the `+`/`-`/space marker column,
which survives horizontal scrolling.

The usual hazard — a syntax highlighter's SGR resets punching holes in a row
background — is already solved in this repo. `appstyles.FillBackground`
re-asserts the background after every reset, and `HasBackgroundBleed` exists
so a test can assert the invariant on a rendered frame. Apply
`FillBackground(rowTint, …)` per line.

Auto-scroll: `viewport.SetYOffset(firstChangeLine - height/2)`, clamped.
`SoftWrap` is off, so line index maps 1:1 to row.

Per CONTRIBUTING, anything that shows up only on screen gets a VHS tape
before it is committed. This phase is that; the focus affordance and the
diff tint are both things to look at rather than reason about.

## Deferred

**Intra-line highlighting.** `udiff.Strings` gives rune-level edits with byte
offsets for a pair of lines, but *pairing* a deleted line with its
replacement is a heuristic no library provides. Pair only on a clean
1-delete/1-insert (or equal counts, in order), and add a similarity guard —
reject the pairing when common prefix plus suffix is under roughly 30% of
the line — so two unrelated lines do not get sprayed with confetti. Degrade
to a plain whole-line tint whenever the heuristic declines, the same
best-effort posture `src/highlight` already takes. Fills `Line.Spans`.

**Hunks-only view.** The panel shows the whole file. A key to collapse to
changed hunks with N lines of context is a natural follow-up, not a
requirement.

## Traps found the hard way

Three things cost real time in phases 0-2 and would cost it again.

**`drive` does not run commands.** The model-level test helper applies
messages only, and this page is a four-message chain: activate →
`GetBackups` → `BackupListMsg` → the list publishes → the preview reads. A
test that stops at the first message renders zero-width panels, so every
assertion about panel contents passes vacuously — which is how the first
version of `TestSelectingABackupReachesThePreview` passed against a list
that published nothing at all. Use `settle` in `src/model/backup_test.go`,
and negative-control any test that asserts on a rendered frame.

**`withComposeLoaded` sets no terminal size.** Without a `WindowSizeMsg` the
panels are zero-width and render nothing. Same failure mode as above,
different cause.

**Benchmark a per-keystroke rebuild before believing it is cheap.** The
viewport approach in Phase 2 looked obviously right and measured 39ms per
cursor move at the store's ceiling. Phase 5 renders diff lines per frame and
is the next place this could bite.

**`AppModel.pages` is a map, so every copy of the model shares its
components.** A test that renders the pre-action model *after* driving the
action renders the post-action state, and a "did this change the frame?"
assertion passes vacuously. Capture the before-frame first. This has now cost
time twice: once on a frame comparison, and once on a test that filtered a list
two earlier statements had already left mid-filter. If a test drives the same
`m` down two paths, the second path starts where the first ended.

**A test that feeds commands back cannot wait on them.** A list's filter input
returns a cursor-blink command that sleeps for the blink interval and then asks
to be run again, so a settle loop that runs every command is an infinite loop
paced at half a second a lap. Both settle helpers now abandon a command that
has not answered in 50ms. Related: **filter keystrokes have to be settled one
at a time.** bubbles narrows the rows through a command, so queueing the whole
term ahead of the first match lands `enter` on a list that has matched nothing,
and bubbles drops an accepted filter with no matches back to unfiltered.

**The rig's screen decoder is not a terminal.** It covers the sequences the
renderer has been *observed* to emit, and a frame it cannot place comes back
as a plausible-looking screen with the wrong content — not as an error. When a
`WaitFor` fails and the screen dump looks garbled or is missing something the
app plainly drew, check `r.Output()` for the raw bytes before believing the
app is at fault.

**Two snapshots inside the same second sort by hash, not by write order.**
`.bak` names start with a UTC timestamp at second granularity, so two
snapshots taken in the same second tie on the prefix and fall back to their
sha8 - which is not write order. Any test that seeds two copies and then
assumes "newest first" matches write order can pick the wrong one; find a
seeded copy by its content hash (`utils.ContentSHA8`) instead of by position.
Phase 4's preview tests hit this on their first run.

**Rebuild before recording.** `make build` writes `cais` into `GOBIN`, and VHS
records whatever binary is there. Recording with a stale binary looks exactly
like the bug you fear most - the feature silently absent in the real app - and
the version line in the footer names the commit it was stamped from, which is
how to catch it. Phase 4's first recording ran `v0.6.0-50-g2306f84` against
commits that had landed three commits earlier.

## Order and why

Phase 0 is independent and should land first — it is the rule the rest of
this work is written against, and it is cheap. Phase 2 is the only phase
that fixes something users hit today, and it does not depend on the diff, so
it should not wait behind it. Phase 1 has to precede 3, 4 and 5 because they
all assume two components. Phases 4 and 5 are the feature.

Every phase is independently shippable and independently green.
