# Backups Page Rework — Implementation Plan

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
- `src/components/backupdiffpanel` — owns the viewport and the rendered
  diff.

Selection moves between them the way Home already does it: the list emits
`SetSelectedBackupMsg` on cursor move, AppModel holds it, both panels render
from it. The list stops reading `.bak` bytes itself; the diff panel (or
AppModel) does.

No behavior change beyond the layout in this phase. Land it green, with the
existing tests ported, before anything else moves.

## Phase 2 — Make the list a real viewport

Fixes defect 2. The list gets a `viewport.Model` with an explicitly
constructed keymap, and a cursor-follow: after a selection move, adjust the
offset so the selected row stays visible. Two lines per row, so the
arithmetic is in rows, not entries — worth a table test at the boundaries
(first row, last row, exactly-fits, one-past-fits).

This is the one phase that fixes a bug users hit today. It is independent of
the diff and of focus, and should land on its own.

## Phase 3 — Focus

`AppModel` gains a `backupsFocus` field, valid only while
`activePage == "Backups"`, and handles `tab` / `shift+tab` scoped to that
page — AppModel does not currently match `tab` at all
(`model/Update.go:466-468` is a tombstone comment where the old handling
was), so nothing is being taken from anywhere else. Handling it in one place
rather than in both panels keeps a single owner for the key, per Phase 0.

The new focus is broadcast to both panels, which route their own keys by the
flag:

| Key | List focused | Diff focused |
| --- | --- | --- |
| `↑` `↓` `j` `k` | move cursor | scroll one line |
| `g` `G`, `home` `end` | first/last entry | top/bottom of file |
| `pgup` `pgdn` `ctrl+u` `ctrl+d` | page the list | page the file |
| `r` | restore | restore |
| `tab` `shift+tab` | → diff | → list |

Focus is shown with the documented tier lift: the focused panel sits on
`BackgroundElevated`, the unfocused on `BackgroundPanel`. `chrome.PanelBg()`
returns elevated unconditionally today (focus having been removed
app-wide), so add a `chrome.PanelBgFor(isFocused bool)` used only by these
two panels rather than changing `PanelBg()`'s signature across every
component.

Also in this phase: drop `Global.Back` from `keys.Active()`'s `"Backups"`
case (defect 3), drop `enter` from `Backup.Restore`, move the viewport
keymaps into `src/keys/Keys.go` beside `ListKeyMap()` — CONTRIBUTING
requires a binding to be declared once, there, or the footer cannot render
from the same source — and add a scroll hint to the footer modeled on
`Files.Scroll`. Fix defect 4 while moving the keymap.

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

Wiring: the diff needs the **live file's** contents for the matching source
(`entry.Source` is `"compose"` or `".env"`). The panel deliberately does not
know which files are loaded — AppModel supplies resolved paths, per the
request/response split. So live contents ride in on the message rather than
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

## Order and why

Phase 0 is independent and should land first — it is the rule the rest of
this work is written against, and it is cheap. Phase 2 is the only phase
that fixes something users hit today, and it does not depend on the diff, so
it should not wait behind it. Phase 1 has to precede 3, 4 and 5 because they
all assume two components. Phases 4 and 5 are the feature.

Every phase is independently shippable and independently green.
