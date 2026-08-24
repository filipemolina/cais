---
title: Theme System
description: How the 13 themes are built from base colors, and the rules that keep them consistent.
---

# Theme System

Every color the app draws with is a field on `appstyles.Theme` (`src/appstyles/Theme.go`), not a hex value scattered through a component. `appstyles.Active` is the one `Theme` in effect; every call site reads it fresh — `appstyles.Active.TextPrimary`, say — rather than caching a color at package init, which is what lets a later switch actually repaint: assign a different registered `Theme` to `Active` and the next frame draws it.

The handful of styles that are more than one field — `appstyles.NormalTitle`, `LogsModal.go`'s `logsModalWrapper`, `model/View.go`'s `errorBannerStyle` — are functions for the same reason, not package-level `var`s: a `var` built at init freezes whichever theme was active when the package loaded.

## The registry

`appstyles.Themes` is the registry: 14 themes — 3 cais themes (cais-dark, cais-dusk, cais-day) plus 11 community schemes (Catppuccin Mocha, Catppuccin Latte, Gruvbox Dark, Gruvbox Light, Tokyo Night, Nord, Dracula, Solarized Dark, Everforest Dark, Rosé Pine, Monokai Pro).

Each theme is built by `appstyles.newTheme` from a handful of base colors — `Accent`, the text/panel/modal bases, `Danger`, the four status colors — with everything else derived by `Lighten`/`Darken`. **Adding a theme is choosing those base colors, not hand-tuning thirty derived ones.**

## The asymmetry that drives every imported palette

`Lighten` is additive (+10/+20/+31 per channel at the standard deltas) and `Darken` is multiplicative (×0.96/×0.92/×0.88). For a dark theme this is a fixed climb independent of the base; for a light theme the steps shrink as the base approaches white.

The consequence for imported color schemes: **set `Panel` to that scheme's deepest background tier** (crust / bg_dim / bg0_hard / sumiInk0 / bg_dark), so the +8% tier (`BackgroundPanel`) lands back on the scheme's signature background. `Modal` must clear `BackgroundElevated` by ≥14 per channel or the modal disappears into the body panels, which render on that same elevated tier.

## The one deliberate exception

`InkOnLight`/`InkOnDark` do not vary with a theme's `Dark` flag, because a status pill's own fill (`StatusRunning` green, `StatusStarting` amber, `StatusError` red) does not vary with the app's theme either — the text that reads legibly on a green pill has to stay dark whichever theme is active, not follow `TextPrimary`, which flips.

With the expanded registry, hard-coding which ink to use on a given fill is no longer survivable — the same call site draws on a `#BC3FBC` magenta in one theme and a `#A7C080` sage in another. The `appstyles.InkOn(fill)` helper picks whichever of the two fixed inks has better contrast on the fill at hand, and `Contrast_test.go` verifies the result clears 4.2:1 on every status pill, the accent chip, and the error banner for every registered theme.

## Background tiers

Sections are separated by background color rather than by borders. The tiers are `Theme` fields, read through `appstyles.Active`:

| Tier | Field | Where |
| --- | --- | --- |
| 1 | terminal default | outside the app — never drawn on |
| 2 | `BackgroundContent` | the frame: header, footer, gutter |
| 3 | `BackgroundPanel` | no painted surface — the mid tier survives as spacing between the frame and the panels |
| 4 | `BackgroundElevated` | the body panels — both, always (see below) |
| — | `ModalBg` | modals, and an active list row — its own register, not derived from the panel tiers |

Both body panels render at tier 4 permanently. The lift that used to mark the focused panel outlived focus itself: with the focus cursor gone there is no second state for a panel to be in, so the old lifted look became the steady state. `chrome.PanelBg()` is the one place that choice lives.

One surface runs the other way: `BackgroundRecessed` sits *below* the panel tier — it is the theme's un-raised `PanelBg` — and is used for insets like the empty-state cards, which read as cut into the panel rather than raised off it.

## Sealing the tiers

A terminal's SGR reset clears the background until the next SGR, and lipgloss closes each styled run with a reset — so any unstyled text later on the same line renders on the terminal's own color. Two rules follow:

1. **Anything that draws text needs an explicit background**, including buttons, cards, and list rows. Components that sit inside a panel take that panel's tier as a parameter instead of picking a tint of their own, so they stay flush with their panel's tier.
2. **Seal innermost-first.** Each tier seals its own region, then the next tier out seals what is left. The outermost seal is the tier-2 pass in `AppModel.View`.

`appstyles.HasBackgroundBleed` is the matching assertion, and `src/model/background_test.go` applies it to fully rendered frames across both pages and their empty, populated, narrow, and error-banner states — once per registered theme, via a `forEachTheme` helper that sets `appstyles.Active` and restores it after.

## The theme picker

`T` (shift+t) opens a modal listing every registered theme, sorted by name, with the active one marked and the cursor starting on it. Cursor movement previews live: `ThemePickerModalModel.Update` detects an index change after the list update and calls `appstyles.SetTheme`, so the entire UI behind the modal repaints on each keystroke. The original theme is captured at construction time, so `Esc` always restores what the user started with. `Enter` applies and persists: the choice is written to `~/.config/cais/config.yaml` via `config.SaveConfig`, and the saved theme is loaded in `main.go` before the program starts.

## Adding a theme

Choose the handful of base colors in `themeParams` (accent, text, panel, modal, danger, the four status colors); `newTheme` derives every other field. Register it in `appstyles.Themes`.

The three cais themes share one set of status and danger colors on purpose: container state is a vocabulary the user should not have to re-learn between them. An imported scheme brings its own, because a cais teal dropped into Gruvbox would read as the one thing on screen that is not Gruvbox; what stays constant there is the *mapping* (green runs, amber starts, red errs), not the hex.