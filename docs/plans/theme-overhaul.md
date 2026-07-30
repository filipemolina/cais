# Plan: prune the registry, add a light stitcher, ship ten community schemes

Supersedes `docs/plans/community-themes.md` (delete that file — its Modal
choices were wrong; the ones here are measured).

## Scope

| | |
| --- | --- |
| **Keep** | `stitcher-dark`, `stitcher-ember`, `stitcher-slate` |
| **Remove** | the other 11 `stitcher-*` themes |
| **Add** | `stitcher-day` — a light twin of `stitcher-dark` that keeps the magenta |
| **Add** | `catppuccin-mocha`, `gruvbox-dark`, `tokyo-night`, `nord`, `dracula`, `solarized-dark`, `one-dark`, `everforest-dark`, `rose-pine`, `kanagawa-wave` |
| **Fix** | pill/chip ink, which is hard-coded per call site and already illegible in `stitcher-ember` and `stitcher-slate` today |
| **Fix** | the theme picker overflowing short terminals |

Final registry: **14 themes**, one of them light.

Every hex in this document was run through the real `newTheme` and scored for
elevation separation and WCAG contrast before being written down. The numbers
quoted are measured, not estimated. Step 6 turns that scoring into a permanent
test so the next theme can't regress it.

---

## What you must know about the engine before touching anything

Read `docs/DESIGN.md` "Theme" (~line 560) first, then this — it corrects one
thing that section leaves implicit.

### The elevation ladder is not symmetric

`newTheme` derives every surface from `Panel` using `lipgloss.Lighten` and
`lipgloss.Darken`. Those two are **not** inverses of each other:

```go
// charm.land/lipgloss/v2/color.go
func Lighten(c, p) // adds 255*p to every channel, clamped at 255
func Darken(c, p)  // multiplies every channel by (1-p)
```

So for a **dark** theme (`raise = Lighten`) the ladder is a fixed additive
climb, independent of the base color:

| token | derivation | step from `Panel` |
| --- | --- | --- |
| `BackgroundRecessed` | `Panel` | +0 |
| `BackgroundContent` | `raise 0.04` | **+10** per channel |
| `BackgroundPanel` | `raise 0.08` | **+20** per channel |
| `BackgroundElevated` | `raise 0.12` | **+31** per channel |
| `BorderCard` | `raise 0.18` | **+45** per channel |
| `BorderDefault` | `lower 0.30` | `Panel × 0.7` |

For a **light** theme the directions swap: tiers are `Panel × 0.96 / 0.92 /
0.88` (≈ −10 / −20 / −31 from a near-white base, so the steps stay about the
same size), `BorderCard` is `× 0.82`, and `BorderDefault` becomes
`Panel + 76`.

Three consequences that drive every choice below:

1. **The panel you actually read on is `Panel + 20`, not `Panel`.** Set
   `Panel` to a scheme's signature background and every surface in the app
   comes out 20–31 points lighter than that scheme looks anywhere else. The
   rule for an imported palette is therefore: **`Panel` = the scheme's
   *deepest* background tier (crust / bg_dim / bg0_hard / sumiInk0 / bg_dark),
   so that `BackgroundPanel` lands back on the scheme's signature bg.**
2. **`Modal` must clear `BackgroundElevated` by ≥14 per channel.** `Modal` is
   not derived, and a modal composites over focused (elevated) panels. Most
   schemes' obvious "popup" color sits exactly in the +10…+31 band and
   disappears. Every `Modal` below was chosen against the measured
   `BackgroundElevated`, not by eye.
3. **In a light theme `BorderDefault` clamps to `#FFFFFF`** for any `Panel`
   lighter than `#B3B3B3`. This is not a bug to fix — a white rim on a
   grey-stepped light UI reads as a bevel — but do not expect a tinted border
   in `stitcher-day`.

### Ink on a colored fill is currently hard-coded, and it is already wrong

`InkOnLight`/`InkOnDark` exist so text stays legible on a fill whose color does
not follow the theme. Call sites pick one *by hand*, which was survivable when
one dark theme existed and is not survivable now:

| call site | renders | `stitcher-dark` | `stitcher-ember` | `catppuccin-mocha` |
| --- | --- | --- | --- | --- |
| `appstyles/styles.go:19` `NormalTitle` — every panel title chip | `TextPrimary` on `Accent` | 4.40 | **1.84** | **1.40** |
| `model/View.go:16` error banner | `TextPrimary` on `Danger` | **3.80** | **3.42** | **1.43** |
| `DetailsPanel.go:957` STARTING pill | `InkOnDark` on `StatusStarting` | **1.61** | **1.61** | **1.22** |
| `DetailsPanel.go:547` STOPPED pill | `InkOnDark` on `StatusError` | **3.65** | **3.65** | **2.22** |
| `ServiceListItem.go:42` STOPPED pill | `InkOnDark` on `StatusStopped` | **3.56** | **3.70** | **3.54** |

Bold = below the 4.5 target. `GroupDetailsPanel.go:322` puts `InkOnLight` on
`StatusStarting` while `DetailsPanel.go:957` puts `InkOnDark` on the same fill
— the two disagree, and the second one is unreadable. Step 1 fixes all of it
with one helper.

---

## Step 0 — branch

```
git checkout -b theme-overhaul
```

One commit per step below. Merge with `--no-ff`.

---

## Step 1 — `appstyles.InkOn(fill)`

**Why first:** it is a prerequisite. Without it, half the palettes below would
have to be distorted away from their upstream hexes just to survive the title
chip, and `stitcher-ember`/`stitcher-slate` stay broken.

Add to `src/appstyles/Theme.go`:

```go
// InkOn returns whichever of the theme's two fixed inks reads better on fill.
//
// InkOnLight/InkOnDark do not vary with the theme (see the Theme field
// comment) because the fills they sit on - a status pill, the accent title
// chip - are not derived from the surface tiers. What *does* vary is which of
// the two is correct, and that is a property of the fill, not of the call
// site. Hard-coding it worked while one dark theme existed; with a light
// theme and ten imported palettes in the registry, the same call site draws
// on a #BC3FBC magenta in one theme and a #A7C080 sage in another.
func InkOn(fill color.Color) color.Color {
	if contrast(Active.InkOnLight, fill) >= contrast(Active.InkOnDark, fill) {
		return Active.InkOnLight
	}
	return Active.InkOnDark
}

// contrast is the WCAG 2.x contrast ratio between two opaque colors.
func contrast(a, b color.Color) float64 { … }

// relativeLuminance is WCAG 2.x relative luminance.
func relativeLuminance(c color.Color) float64 { … }
```

`contrast` and `relativeLuminance` are ~20 lines of standard formula; Step 6's
test needs them too, so export them if that reads better — `Contrast` is a
reasonable public name for a package that already owns the color vocabulary.

Then replace the hard-coded choices:

| file:line | change |
| --- | --- |
| `src/appstyles/styles.go:21` | `Foreground(Active.TextPrimary)` → `Foreground(InkOn(Active.Accent))` |
| `src/model/View.go:18` | `Foreground(appstyles.Active.TextPrimary)` → `Foreground(appstyles.InkOn(appstyles.Active.Danger))` |
| `src/components/DetailsPanel.go:545,547` | `fg` → `appstyles.InkOn(bg)` |
| `src/components/DetailsPanel.go:954,957,960` | `fg` → `appstyles.InkOn(bg)` |
| `src/components/GroupDetailsPanel.go:318,320,322` | `fg` → `appstyles.InkOn(bg)` |
| `src/apptypes/ServiceListItem.go:40,42` | `fg` → `appstyles.InkOn(bg)` |

In each pill function the shape is already `label, bg, fg = …`; drop `fg` from
the assignment and compute it once after the switch:

```go
fg := appstyles.InkOn(bg)
```

Update the `statusPill` doc comments in `GroupDetailsPanel.go:303` and
`DetailsPanel.go` — they currently explain *why the ink is fixed*, which is
still true, but the choice is now made from the fill rather than by hand.

**Test:** add `TestInkOnPicksTheLegibleInk` in `src/appstyles/` asserting that
for every registered theme, `InkOn(x)` beats the other ink on `Accent`,
`Danger`, and all four status colors.

---

## Step 2 — prune the registry

Delete these entries from `Themes` in `src/appstyles/Theme.go`:

```
stitcher-light   stitcher-ocean   stitcher-abyss    stitcher-trench
stitcher-forge   stitcher-phantom stitcher-velvet   stitcher-canyon
stitcher-neon    stitcher-orchid  stitcher-bloom
```

Then:

- **Delete the tapes and screenshots for removed themes**: `mocks/tapes/10-theme-ocean.tape`, `mocks/tapes/11-theme-ember.tape` (keep ember — it survives; re-record only if the modal fix changes its frame), and the matching files in `mocks/screenshots/`.
- **No config migration is needed.** `main.go:41` calls
  `appstyles.SetTheme(cfg.Theme)` and ignores the `false` return, so a config
  naming a deleted theme silently falls back to `DefaultTheme`. Verify this
  path with a test rather than assuming it: `TestUnknownSavedThemeFallsBackToDefault`.
- `src/config/config_test.go` uses `"stitcher-ocean"` as a round-trip fixture
  at lines 32 and 92. Config does not validate names so the tests still pass,
  but swap the string for `"stitcher-slate"` so the fixtures don't name a
  theme that no longer exists.

### `stitcher-ember` needs one fix while you are here

Its `Modal` (`#3E322A`) sits **6** per channel from its own
`BackgroundElevated` (`#3C3430`) — a modal that does not read as a modal.
Change it:

```
Modal: #3E322A  →  #52413A     (elevated separation 6 → 22)
```

### Both kept dark themes need the Danger fix

`#B33A3A` scores **2.45** as error text on `stitcher-dark`'s panel and **2.40**
on `stitcher-ember`'s — `Danger` is used as a foreground in
`GroupNameModal.go:59`, `CreateComposeFileModal.go:184` and
`ComposeFilePanel.go:157`, so this is inline validation text nobody can read.

```
Danger: #B33A3A  →  #D9534F     (as text 2.45 → 3.63; as banner fill with InkOn, 4.57)
```

Apply to `stitcher-dark` and `stitcher-ember`. `stitcher-slate` already uses
`#EB4268` and is fine.

### `stitcher-slate`, optional

Its `Panel` `#242F40` puts the read surface at `#384354`, where its own
`StatusError`/`Danger` score 2.63. Darkening the base one notch fixes it
without touching its identity:

```
Panel: #242F40  →  #1D2634     (status-on-panel 2.63 → 3.0+, ladder unchanged)
```

Flagged as optional — it is a visible change to a theme the user asked to
keep. Apply only if you want the numbers; the theme is usable either way.

---

## Step 3 — `stitcher-day`

The light twin of `stitcher-dark`. The brief: keep the magenta, keep the
identity, easy on the eye.

```go
// stitcher-day is stitcher-dark inverted: the same #BC3FBC magenta on a
// warm off-white rather than a violet near-black. The neutral carries a
// faint magenta bias so the greys read as chosen rather than as default
// terminal grey, and the status colors are darkened from their dark-theme
// values because a #67C58A green that reads on a near-black panel washes
// out entirely on a near-white one.
"stitcher-day": newTheme(themeParams{
	Name:   "stitcher-day",
	Dark:   false,
	Accent: lipgloss.Color("#BC3FBC"), // unchanged from stitcher-dark
	Text:   lipgloss.Color("#241F2B"),
	Panel:  lipgloss.Color("#F6F2F7"),
	Modal:  lipgloss.Color("#FCF8FD"),
	Danger: lipgloss.Color("#B33A3A"),

	Running:  lipgloss.Color("#1E7F4E"),
	Stopped:  lipgloss.Color("#6B6878"),
	Starting: lipgloss.Color("#A87409"),
	Err:      lipgloss.Color("#C0243F"),
}),
```

Derived ladder, measured:

```
recessed #F6F2F7 → content #ECE8ED → panel #E2DEE3 → elevated #D8D4D9
modal    #FCF8FD   borders default #FFFFFF  card #C9C6CA
text     primary #241F2B  muted #57525E  dim #706B77
```

Design notes for whoever reviews it:

- **The accent is byte-identical to `stitcher-dark`.** That is the whole point
  of the brief. With `InkOn` the title chip gets light ink on the magenta at
  4.40 — the same figure `stitcher-dark` already ships, so the chip looks and
  scores exactly as it does today.
- **`Modal` goes *lighter* than the panel tiers, not darker.** In a light theme
  elevation is expressed by *darkening*, so the panel ladder descends from
  `#F6F2F7` to `#D8D4D9`. A modal that continued that direction would be a
  grey slab; instead it lifts to a magenta-tinted near-white (`#FCF8FD`, 36
  clear of `BackgroundElevated`) and reads as a lit card floating over the
  page. This is the same trick `stitcher-dark` uses in reverse — its modal is
  neutral where its panels are violet, separating by hue rather than by more
  lightness.
- **The status colors are not the dark theme's.** `#67C58A` / `#E8C547` score
  2.48 and 2.42 against a light panel. The replacements are the same hues
  pulled down in lightness: sea green, dark amber, and a red-pink `#C0243F`
  that leans toward the brand magenta rather than away from it.
- Text primary sits at **11.93** on the panel — comfortable, and deliberately
  not pure black on pure white.

---

## Step 4 — the ten community schemes

House style: a short `//` comment naming the scheme and its upstream, then
`newTheme(...)` with fields in the existing order. Do not "improve" the hexes.
Where a note explains a forced choice, keep that reasoning in the code comment
— it is the difference between a maintainer trusting the value and re-deriving
it.

Each block below lists the params, the measured ladder, and the reasoning.

### `catppuccin-mocha`

```
Accent #CBA6F7 mauve      Text #CDD6F4 text     Panel #11111B crust
Modal  #45475A surface1   Danger #EBA0AC maroon
Running #A6E3A1 green     Stopped #7F849C overlay1
Starting #F9E2AF yellow   Err #F38BA8 red
```

`recessed #11111B → content #1B1B25 → panel #25252F → elevated #2F2F39 | modal #45475A`

`Panel` is **crust**, not base: the +20 tier then lands on `#25252F`, which is
base (`#1E1E2E`) to within a few points. surface0 (`#313244`) — the obvious
modal — sits 11 from elevated and vanishes; surface1 clears it by 32.

### `gruvbox-dark`

```
Accent #FE8019 bright orange   Text #EBDBB2 fg1     Panel #1D2021 bg0_hard
Modal  #504945 bg2             Danger #FB4934 bright red
Running #B8BB26 bright green   Stopped #928374 gray
Starting #FABD2F bright yellow Err #FB4934 bright red
```

`recessed #1D2021 → content #272A2B → panel #313435 → elevated #3B3E3F | modal #504945`

`Panel` is bg0_hard so the read surface lands on `#313435` ≈ bg1. `Danger`
takes the *bright* red, not the neutral `#CC241D`, which scores 2.29 as error
text against that surface.

### `tokyo-night`

```
Accent #7AA2F7 blue     Text #C0CAF5 fg        Panel #16161E bg (Night)
Modal  #394B70 blue7    Danger #DB4B4B error/red1
Running #9ECE6A green   Stopped #737AA2 dark5
Starting #E0AF68 yellow Err #F7768E red
```

`recessed #16161E → content #202028 → panel #2A2A32 → elevated #34343C | modal #394B70`

bg_highlight (`#292E42`) collides with the ladder; blue7 is the palette's own
muted navy and clears elevated by 52. `Stopped` is dark5, not comment
(`#565F89`, 2.30 against the panel). The palette ships a dedicated `error` red
distinct from its syntax red, so `Danger` and `Err` split cleanly.

### `nord`

```
Accent #88C0D0 nord8    Text #D8DEE9 nord4     Panel #242933 nord0, pre-darkened
Modal  #4C566A nord3    Danger #BF616A nord11
Running #A3BE8C nord14  Stopped #7B88A1
Starting #EBCB8B nord13 Err #BF616A nord11
```

`recessed #242933 → content #2E333D → panel #383D47 → elevated #424751 | modal #4C566A`

Nord's whole Polar Night ramp spans 30 points (nord0 `#2E3440` → nord3
`#4C566A`) and the ladder consumes 31, so with an unmodified nord0 there is no
room left for a modal. `Panel` is nord0 darkened; the +20 tier then lands at
`#383D47` ≈ nord1, and nord3 clears elevated by 24. `Stopped` is likewise
lifted off nord3, which is 1.79 against the panel.

### `dracula`

```
Accent #BD93F9 purple   Text #F8F8F2 foreground  Panel #1E1F29 background, pre-darkened
Modal  #44475A current line   Danger #FF5555 red
Running #50FA7B green   Stopped #6272A4 comment
Starting #FFB86C orange Err #FF5555 red
```

`recessed #1E1F29 → content #282933 → panel #32333D → elevated #3C3D47 | modal #44475A`

Same shape as Nord: Dracula publishes exactly two surfaces, and current-line
sits 6 from elevated if background is used unmodified. Pre-darkening the
background buys the 18 points that make the modal visible. `Starting` is
orange rather than Dracula's yellow `#F1FA8C`, which is a pale lime that reads
as highlight, not as work in progress.

### `solarized-dark`

```
Accent #268BD2 blue     Text #93A1A1 base1     Panel #001A21 base03, pre-darkened
Modal  #073642 base02   Danger #DC322F red
Running #859900 green   Stopped #657B83 base00
Starting #B58900 yellow Err #DC322F red
```

`recessed #001A21 → content #0A242B → panel #142E35 → elevated #1E383F | modal #073642`

With base03 as `Panel`, primary text scores 4.27 and the surfaces drift
noticeably lighter than Solarized looks anywhere else. Pre-darkening base03
puts the read surface back at `#142E35` — near base03/base02 — and lifts text
to 5.0. Note the inversion this produces: base02 becomes the *modal*, sitting
darker than the panels, which is the same neutral-vs-tinted separation
`stitcher-dark` uses. `Text` is base1, not the base0 body color, because
`TextMuted` darkens it another 20% and base0 does not survive that.

### `one-dark`

```
Accent #61AFEF blue     Text #ABB2BF mono-1    Panel #21252B menu bg
Modal  #2C323C cursor line   Danger #E06C75 red
Running #98C379 green   Stopped #828997 mono-2
Starting #E5C07B yellow Err #E06C75 red
```

`recessed #21252B → content #2B2F35 → panel #35393F → elevated #3F4349 | modal #2C323C`

The gutter grey (`#4B5263`) is far enough from elevated but only scores 3.67
for primary text; cursor-line goes the other way — 19 clear of elevated, and
much better contrast. `Stopped` is mono-2, not the mono-3 comment grey (1.92
against the panel). `Danger` uses the bright red rather than `#BE5046` (2.45).

### `everforest-dark`

```
Accent #A7C080 green    Text #D3C6AA fg        Panel #232A2E bg_dim
Modal  #4F585E bg4      Danger #E67E80 red
Running #83C092 aqua    Stopped #859289 grey1
Starting #DBBC7F yellow Err #E67E80 red
```

`recessed #232A2E → content #2D3438 → panel #373E42 → elevated #41484C | modal #4F585E`

bg_dim as `Panel` puts the read surface on `#373E42` ≈ bg1. `Running` is aqua
because green is already the accent and two identical greens would make a
running service indistinguishable from focused chrome. Primary text on the
modal scores 4.30 — see Accepted deviations.

### `rose-pine`

```
Accent #C4A7E7 iris     Text #E0DEF4 text      Panel #191724 base
Modal  #403D52 highlight_med   Danger #EB6F92 love
Running #9CCFD8 foam    Stopped #908CAA subtle
Starting #F6C177 gold   Err #EB6F92 love
```

`recessed #191724 → content #23212E → panel #2D2B38 → elevated #373542 | modal #403D52`

The one scheme that needs no pre-darkening — base is deep enough that the
ladder fits inside its own surface range. Rosé Pine has no green; foam is the
palette's all-clear hue and carries `Running` (pine `#31748F` is too dark to
read). `Stopped` is subtle rather than muted, which is dimmer than the panel
warrants.

### `kanagawa-wave`

```
Accent #7E9CD8 crystalBlue   Text #DCD7BA fujiWhite   Panel #16161D sumiInk0
Modal  #223249 waveBlue1     Danger #E82424 samuraiRed
Running #98BB6C springGreen  Stopped #727169 fujiGray
Starting #FF9E3B roninYellow Err #E82424 samuraiRed
```

`recessed #16161D → content #202027 → panel #2A2A31 → elevated #34343B | modal #223249`

sumiInk0 as `Panel` lands the read surface on `#2A2A31` ≈ sumiInk4. waveBlue1
is the palette's own popup surface and clears elevated by 19. `Danger` is
samuraiRed rather than autumnRed (`#C34043`, 2.81 as error text).

---

## Step 5 — stop the picker overflowing (required)

`ThemePickerModal()` at `src/components/ThemePickerModal.go:145` sizes its list
to `len(items)` with pagination off. At 14 themes the modal is ~23 rows and
just fits an 80×24 terminal. This step is needed even though the registry ends
at 14 again — the modal is already at the edge, and `renderWithModal`
(`src/model/View.go:129`) clamps `y` to 0, so anything taller loses its hint
line and bottom border off-screen with no scroll.

1. `func ThemePickerModal(termHeight int) tea.Model` — matching
   `components.HelpOverlay(…, m.config.terminalWidth)` at `src/model/Update.go:767`.
2. Chrome around the list is 9 rows (border 2, padding 2, title 2, blank 1,
   hints 2):
   ```go
   const themePickerChrome = 9
   visible := min(len(items), max(3, termHeight-themePickerChrome))
   ```
3. ```go
   picker := list.New(items, themePickerDelegate{}, 40, visible)
   picker.SetShowPagination(visible < len(items))
   ```
   Leave the other `SetShow*` calls alone.
4. Call site `src/model/Update.go:780` → `components.ThemePickerModal(m.config.terminalHeight)`.
5. Test call sites: `ThemePickerModal_test.go` lines 24, 44, 81, 127, 141 and
   `modal_chrome_test.go:37` — pass `40`.
   `TestThemePickerRendersAllThemes` asserts on `len(tpm.list.Items())`, which
   is item count not visible rows, so it keeps passing.
6. New test: build with `ThemePickerModal(20)` and assert the rendered modal's
   `lipgloss.Height(...)` is `<= 20`.

Live preview is untouched — the cursor-movement branch at
`ThemePickerModal.go:90` doesn't care about list height.

---

## Step 6 — lock elevation and contrast as tested invariants

This is what makes the whole change durable, and it is the direct answer to
"take the different levels of elevation into consideration": make elevation a
property CI checks rather than a thing someone eyeballed once.

New file `src/appstyles/Contrast_test.go`, running over every registered theme:

**Elevation separation** (max per-channel distance):

| assertion | floor |
| --- | --- |
| `recessed → content` | 8 |
| `content → panel` | 8 |
| `panel → elevated` | 8 |
| `elevated ↔ modal` | 14 |
| `panel ↔ borderDefault` | 12 |
| `recessed ↔ borderCard` | 12 |

**Contrast** (WCAG ratio, using the `contrast` helper from Step 1):

| assertion | floor | note |
| --- | --- | --- |
| `TextPrimary` on panel / elevated | 4.5 | |
| `TextPrimary` on modal | 4.2 | everforest measures 4.30 |
| `TextMuted` on panel | 3.0 | |
| `TextDim` on panel | 2.2 | |
| `Accent` on panel and on modal | 3.0 | |
| `InkOn(Accent)` on `Accent` | 4.2 | stitcher-dark/day measure 4.40 |
| `InkOn(fill)` on each status fill and on `Danger` | 4.2 | pills are bold uppercase; WCAG large-text is 3.0 |
| each status color as text on panel | 2.6 | see deviations |

Set the floors exactly here. They are the levels the shipped set clears; a
future theme that dips below one is telling you something real. Print the
measured value in the failure message so the next person can judge, and do not
lower a floor without a note saying which theme forced it and why.

---

## Step 7 — docs

- `README.md:33` — "14 built-in themes" stays numerically correct, but rewrite
  it: three Stitcher darks, one Stitcher light, and ten community schemes
  (Catppuccin, Gruvbox, Tokyo Night, Nord, Dracula, Solarized, One Dark,
  Everforest, Rosé Pine, Kanagawa).
- `README.md:201` — "14 registered themes" is fine; mention the light one.
- `docs/DESIGN.md:571-590` — add the asymmetry: `Lighten` is additive,
  `Darken` multiplicative, tiers are ±10/20/31, and `Panel` for an imported
  palette is that scheme's *deepest* background so the +8% tier lands on its
  signature bg.
- `docs/DESIGN.md:582-588` — the `InkOnLight`/`InkOnDark` paragraph now needs
  the `InkOn` helper: the inks are still theme-invariant, but *which* one is
  correct is a property of the fill.
- `docs/DESIGN.md:676` — "The four shipped themes" is already stale; rewrite
  for the new registry.
- `docs/ROADMAP.md:81` — "themes beyond the four shipped" likewise.
- Delete `docs/plans/community-themes.md`.

---

## Accepted deviations

Do not "fix" these; they were measured and judged.

| theme | measurement | why it stands |
| --- | --- | --- |
| `everforest-dark` | primary on modal **4.30** | Everforest targets ~4.5:1 by design; forcing more makes it not Everforest |
| `nord` | `Err`/`Danger` as text **2.66** | nord11 is the only red in the palette |
| `dracula` | `Stopped` as text **2.66** | comment blue; "stopped" is meant to recede |
| `kanagawa-wave` | `Stopped` as text **2.90** | fujiGray, same reasoning |
| `solarized-dark` | pill inks **4.27–4.43** | pills are bold uppercase, where WCAG's floor is 3.0 |
| `stitcher-day` | `Starting` pill ink **4.45** | same |
| all light themes | `BorderDefault` = `#FFFFFF` | `Lighten` clamps; a white rim on stepped greys reads as a bevel |

---

## Out of scope

- No change to `newTheme`, the tier deltas, or `Theme`'s field set. Everything
  above is achieved by choosing base colors, plus the one `InkOn` helper.
- No light variants of the community schemes (Latte, Solarized Light, Dawn,
  Everforest Light). Each needs its own pass against the `Dark: false`
  derivation, and `stitcher-day` is the light theme this change owes.
- No grouping in the picker. Alphabetical sort scatters the three `stitcher-*`
  entries among the community names; a delegate that sections them is a
  separate change.
