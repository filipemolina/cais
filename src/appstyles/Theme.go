package appstyles

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme is every color the app draws with, resolved to concrete values. One
// field per semantic token, so a registered Theme is the app's complete
// visual vocabulary - no component builds a color of its own.
//
// A Theme is inert data, not a service: nothing here reads the terminal or
// does I/O. Active (below) is the one Theme in effect, and everything that
// draws reads it fresh on every render rather than caching a color at
// package init - see the package-level styles in styles.go for why that
// distinction matters. That is the whole of what lets a later switch repaint
// the app: assign a different registered Theme to Active and the next frame
// draws it.
type Theme struct {
	Name string
	// Dark says which way the tiers derive: a dark theme raises a surface's
	// attention by lightening it, a light theme by darkening it. See
	// newTheme.
	Dark bool

	// Accent is the brand color: focus, the wordmark, the active tab, title
	// chips. It does not vary with Dark - see newTheme.
	Accent color.Color

	// Text tiers, most to least emphasis.
	TextPrimary color.Color
	TextMuted   color.Color
	TextDim     color.Color

	// PanelBg is the base surface color every background tier below derives
	// from, and is also BackgroundRecessed's value directly.
	PanelBg color.Color
	// BackgroundContent/Panel/Elevated are tiers 2/3/4 in docs/DESIGN.md's
	// "Background tiers, and sealing them": the frame, an unfocused panel,
	// and a focused panel.
	BackgroundContent  color.Color
	BackgroundPanel    color.Color
	BackgroundElevated color.Color
	// BackgroundRecessed sits *below* the panel tier, for insets like the
	// empty-state cards - see docs/DESIGN.md. Equal to PanelBg by
	// construction: both are the tier ladder's un-raised base.
	BackgroundRecessed color.Color
	// ModalBg is the surface every modal - and an active list row - is drawn
	// on: a distinct register from the panel tiers, not derived from
	// PanelBg.
	ModalBg color.Color

	// BorderDefault rims an ordinary panel; in a dark theme it is darker
	// than PanelBg so it all but disappears against a *recessed* fill.
	// BorderCard rims a recessed surface, so it has to go the other way -
	// lighter than PanelBg in a dark theme - or the rim vanishes into what
	// it is meant to outline. See docs/DESIGN.md.
	BorderDefault color.Color
	BorderCard    color.Color

	// Status tiers reflect one container or group's own state, and do not
	// vary with Dark: a "running" dot is the same green whichever theme is
	// active. See InkOnLight/InkOnDark for the text that sits on top of one.
	StatusRunning  color.Color
	StatusStopped  color.Color
	StatusStarting color.Color
	StatusError    color.Color

	// Danger is app-level alert chrome - the error banner, an inline
	// validation message - a different concept from StatusError (one
	// service's own state) even though earlier code used one hex for both
	// by coincidence.
	Danger color.Color

	// InkOnLight and InkOnDark are deliberately theme-invariant: a status
	// pill's fill (StatusRunning green, StatusStarting amber, StatusError
	// red/pink) is the same hue whichever theme is active, so the text that
	// reads legibly on it can't follow Dark either - a bright pill needs
	// dark ink and a dark pill needs light ink regardless of the *app's*
	// theme. See GroupDetailsPanel.go's statusPill.
	InkOnLight color.Color
	InkOnDark  color.Color
}

// themeParams are the handful of base colors newTheme builds a Theme from.
// Everything else is derived - see newTheme.
type themeParams struct {
	Name string
	Dark bool

	Accent color.Color
	Text   color.Color
	Panel  color.Color
	Modal  color.Color
	Danger color.Color

	Running, Stopped, Starting, Err color.Color
}

// newTheme derives a full Theme from a handful of base colors, so adding a
// theme is picking a handful of hex values rather than hand-tuning thirty.
//
// raise/lower pick Lighten or Darken based on Dark: a dark theme raises a
// surface's attention by lightening it (further from the near-black base,
// toward the light text sitting on it); a light theme raises attention by
// darkening it (further from the near-white base, toward the dark text
// sitting on it). Both themes apply the same deltas, so the tiers stay
// proportional to each other; only the direction flips.
func newTheme(p themeParams) Theme {
	raise, lower := lipgloss.Lighten, lipgloss.Darken
	if !p.Dark {
		raise, lower = lower, raise
	}

	return Theme{
		Name: p.Name,
		Dark: p.Dark,

		Accent: p.Accent,

		TextPrimary: p.Text,
		TextMuted:   lower(p.Text, 0.2),
		TextDim:     lower(p.Text, 0.3),

		PanelBg:            p.Panel,
		BackgroundContent:  raise(p.Panel, 0.04),
		BackgroundPanel:    raise(p.Panel, 0.08),
		BackgroundElevated: raise(p.Panel, 0.12),
		BackgroundRecessed: p.Panel,
		ModalBg:            p.Modal,

		BorderDefault: lower(p.Panel, 0.3),
		BorderCard:    raise(p.Panel, 0.18),

		StatusRunning:  p.Running,
		StatusStopped:  p.Stopped,
		StatusStarting: p.Starting,
		StatusError:    p.Err,

		Danger: p.Danger,

		// Fixed regardless of p.Dark - see the Theme field comment.
		InkOnLight: lipgloss.Color("#151520"),
		InkOnDark:  lipgloss.Color("#FAFAFA"),
	}
}

// DefaultTheme is the theme a fresh AppModel starts with.
const DefaultTheme = "stitcher-dark"

// Themes is the registry a theme picker (post-alpha, see docs/ROADMAP.md)
// will choose from. Every entry is built through newTheme rather than a bare
// struct literal, so a registered theme can't leave a field zero-valued the
// way a hand-written literal could - which matters here because a nil
// color.Color renders as no SGR at all, i.e. a background-bleed bug. See
// src/appstyles/Theme_test.go and src/appstyles/Background_test.go.
var Themes = map[string]Theme{
	"stitcher-dark": newTheme(themeParams{
		Name:   "stitcher-dark",
		Dark:   true,
		Accent: lipgloss.Color("#BC3FBC"),
		Text:   lipgloss.Color("#FAFAFA"),
		Panel:  lipgloss.Color("#151520"),
		Modal:  lipgloss.Color("#282828"),
		Danger: lipgloss.Color("#B33A3A"),

		Running:  lipgloss.Color("#67C58A"),
		Stopped:  lipgloss.Color("#858392"),
		Starting: lipgloss.Color("#E8C547"),
		Err:      lipgloss.Color("#EB4268"),
	}),

	// stitcher-light closes the "unusable on a light terminal" risk: today's
	// dark palette is the only one that has ever existed. Accent, the status
	// colors and Danger are shared with stitcher-dark on purpose - they are
	// recognizable, saturated colors that read on both a near-black and a
	// near-white base - only the surfaces and text invert.
	"stitcher-light": newTheme(themeParams{
		Name:   "stitcher-light",
		Dark:   false,
		Accent: lipgloss.Color("#BC3FBC"),
		Text:   lipgloss.Color("#1B1A22"),
		Panel:  lipgloss.Color("#EDEBF5"),
		Modal:  lipgloss.Color("#B2B0BE"),
		Danger: lipgloss.Color("#B33A3A"),

		Running:  lipgloss.Color("#67C58A"),
		Stopped:  lipgloss.Color("#858392"),
		Starting: lipgloss.Color("#E8C547"),
		Err:      lipgloss.Color("#EB4268"),
	}),

	// stitcher-ocean is a dark theme with a cool blue base and a teal
	// accent. Status colors and Danger are shared with the other themes
	// for the same reason as stitcher-light: they are recognizable hues
	// that read on any base.
	"stitcher-ocean": newTheme(themeParams{
		Name:   "stitcher-ocean",
		Dark:   true,
		Accent: lipgloss.Color("#5BA8C8"),
		Text:   lipgloss.Color("#E8F0F8"),
		Panel:  lipgloss.Color("#111B26"),
		Modal:  lipgloss.Color("#2A3A4A"),
		Danger: lipgloss.Color("#B33A3A"),

		Running:  lipgloss.Color("#67C58A"),
		Stopped:  lipgloss.Color("#7A8A96"),
		Starting: lipgloss.Color("#E8C547"),
		Err:      lipgloss.Color("#EB4268"),
	}),

	// stitcher-ember is a dark theme with a warm brown-black base and an
	// amber accent. The same shared status/danger colors keep the
	// container state vocabulary consistent across themes.
	"stitcher-ember": newTheme(themeParams{
		Name:   "stitcher-ember",
		Dark:   true,
		Accent: lipgloss.Color("#E8A44A"),
		Text:   lipgloss.Color("#F5EDE4"),
		Panel:  lipgloss.Color("#1E1612"),
		Modal:  lipgloss.Color("#3E322A"),
		Danger: lipgloss.Color("#B33A3A"),

		Running:  lipgloss.Color("#67C58A"),
		Stopped:  lipgloss.Color("#8A8078"),
		Starting: lipgloss.Color("#E8C547"),
		Err:      lipgloss.Color("#EB4268"),
	}),

	// stitcher-abyss is a dark theme inspired by the deep ocean at night:
	// a near-black violet base, muted teal text, and a luminous mint
	// accent that glows like bioluminescence.
	"stitcher-abyss": newTheme(themeParams{
		Name:   "stitcher-abyss",
		Dark:   true,
		Accent: lipgloss.Color("#d0f4ea"),
		Text:   lipgloss.Color("#a0b8c0"),
		Panel:  lipgloss.Color("#02010a"),
		Modal:  lipgloss.Color("#22007c"),
		Danger: lipgloss.Color("#0d00a4"),

		Running:  lipgloss.Color("#67C58A"),
		Stopped:  lipgloss.Color("#829399"),
		Starting: lipgloss.Color("#E8C547"),
		Err:      lipgloss.Color("#EB4268"),
	}),

	// stitcher-trench channels the deep sea: a dark navy base with vivid
	// cyan accents, like light piercing through black water.
	"stitcher-trench": newTheme(themeParams{
		Name:   "stitcher-trench",
		Dark:   true,
		Accent: lipgloss.Color("#0094c6"),
		Text:   lipgloss.Color("#78d8e8"),
		Panel:  lipgloss.Color("#000022"),
		Modal:  lipgloss.Color("#001242"),
		Danger: lipgloss.Color("#EB4268"),

		Running:  lipgloss.Color("#67C58A"),
		Stopped:  lipgloss.Color("#7A8A96"),
		Starting: lipgloss.Color("#E8C547"),
		Err:      lipgloss.Color("#EB4268"),
	}),

	// stitcher-forge evokes a blacksmith's workshop: charcoal stone
	// surfaces warmed by amber highlights and a coral spark.
	"stitcher-forge": newTheme(themeParams{
		Name:   "stitcher-forge",
		Dark:   true,
		Accent: lipgloss.Color("#f4b860"),
		Text:   lipgloss.Color("#fff0e8"),
		Panel:  lipgloss.Color("#32373b"),
		Modal:  lipgloss.Color("#384848"),
		Danger: lipgloss.Color("#c83e4d"),

		Running:  lipgloss.Color("#67C58A"),
		Stopped:  lipgloss.Color("#858392"),
		Starting: lipgloss.Color("#E8C547"),
		Err:      lipgloss.Color("#EB4268"),
	}),

	// stitcher-phantom is a dusk-themed palette: deep plum surfaces, soft
	// blue text, and a cool mint accent like moonlight on stone.
	"stitcher-phantom": newTheme(themeParams{
		Name:   "stitcher-phantom",
		Dark:   true,
		Accent: lipgloss.Color("#ebf5ee"),
		Text:   lipgloss.Color("#e0ecf5"),
		Panel:  lipgloss.Color("#542344"),
		Modal:  lipgloss.Color("#301a28"),
		Danger: lipgloss.Color("#d8bfaa"),

		Running:  lipgloss.Color("#67C58A"),
		Stopped:  lipgloss.Color("#858392"),
		Starting: lipgloss.Color("#E8C547"),
		Err:      lipgloss.Color("#EB4268"),
	}),

	// stitcher-velvet is a light theme with a silver base, rich rose
	// accents, and deep aubergine for contrast - opulent yet refined.
	"stitcher-velvet": newTheme(themeParams{
		Name:   "stitcher-velvet",
		Dark:   false,
		Accent: lipgloss.Color("#893168"),
		Text:   lipgloss.Color("#1a0c18"),
		Panel:  lipgloss.Color("#eaeaea"),
		Modal:  lipgloss.Color("#c9c0d5"),
		Danger: lipgloss.Color("#893168"),

		Running:  lipgloss.Color("#67C58A"),
		Stopped:  lipgloss.Color("#858392"),
		Starting: lipgloss.Color("#E8C547"),
		Err:      lipgloss.Color("#EB4268"),
	}),

	// stitcher-canyon draws from desert mesas: warm terracotta earth tones
	// with a coral accent and soft peach highlights. Panel and modal were
	// darkened from the source palette to give text enough contrast.
	"stitcher-canyon": newTheme(themeParams{
		Name:   "stitcher-canyon",
		Dark:   true,
		Accent: lipgloss.Color("#ea9580"),
		Text:   lipgloss.Color("#f5ddd5"),
		Panel:  lipgloss.Color("#1e201d"),
		Modal:  lipgloss.Color("#101210"),
		Danger: lipgloss.Color("#a37774"),

		Running:  lipgloss.Color("#67C58A"),
		Stopped:  lipgloss.Color("#858392"),
		Starting: lipgloss.Color("#E8C547"),
		Err:      lipgloss.Color("#EB4268"),
	}),

	// stitcher-neon is an electric jungle: a pitch-black base lit by vivid
	// tropical greens and warm amber - high energy, high contrast.
	"stitcher-neon": newTheme(themeParams{
		Name:   "stitcher-neon",
		Dark:   true,
		Accent: lipgloss.Color("#00a878"),
		Text:   lipgloss.Color("#d8f1a0"),
		Panel:  lipgloss.Color("#0b0500"),
		Modal:  lipgloss.Color("#2a1810"),
		Danger: lipgloss.Color("#fe5e41"),

		Running:  lipgloss.Color("#67C58A"),
		Stopped:  lipgloss.Color("#858392"),
		Starting: lipgloss.Color("#f3c178"),
		Err:      lipgloss.Color("#EB4268"),
	}),

	// stitcher-orchid pairs dusty mauves with an electric cyan pop - like
	// wildflowers blooming under a stormy sky.
	"stitcher-orchid": newTheme(themeParams{
		Name:   "stitcher-orchid",
		Dark:   true,
		Accent: lipgloss.Color("#14bdeb"),
		Text:   lipgloss.Color("#ecd0da"),
		Panel:  lipgloss.Color("#241715"),
		Modal:  lipgloss.Color("#402a2c"),
		Danger: lipgloss.Color("#703d57"),

		Running:  lipgloss.Color("#67C58A"),
		Stopped:  lipgloss.Color("#957186"),
		Starting: lipgloss.Color("#E8C547"),
		Err:      lipgloss.Color("#EB4268"),
	}),

	// stitcher-bloom is a dark volcanic palette: smoldering reds and ambers
	// rising from a deep charcoal base, like embers in twilight.
	"stitcher-bloom": newTheme(themeParams{
		Name:   "stitcher-bloom",
		Dark:   true,
		Accent: lipgloss.Color("#d58936"),
		Text:   lipgloss.Color("#fffb46"),
		Panel:  lipgloss.Color("#3c1518"),
		Modal:  lipgloss.Color("#69140e"),
		Danger: lipgloss.Color("#a44200"),

		Running:  lipgloss.Color("#67C58A"),
		Stopped:  lipgloss.Color("#858392"),
		Starting: lipgloss.Color("#E8C547"),
		Err:      lipgloss.Color("#EB4268"),
	}),

	// stitcher-slate is a refined dark theme with golden accents on a blue-
	// black base - understated elegance with a warm metallic shimmer.
	"stitcher-slate": newTheme(themeParams{
		Name:   "stitcher-slate",
		Dark:   true,
		Accent: lipgloss.Color("#cca43b"),
		Text:   lipgloss.Color("#e5e5e5"),
		Panel:  lipgloss.Color("#242f40"),
		Modal:  lipgloss.Color("#363636"),
		Danger: lipgloss.Color("#EB4268"),

		Running:  lipgloss.Color("#67C58A"),
		Stopped:  lipgloss.Color("#858392"),
		Starting: lipgloss.Color("#E8C547"),
		Err:      lipgloss.Color("#EB4268"),
	}),
}

// Active is the one Theme in effect. Everything that draws reads it fresh
// each render - see the Theme doc comment - so assigning a different
// registered Theme here and re-rendering is the whole of what a theme switch
// needs to do.
var Active = Themes[DefaultTheme]

// SetTheme assigns a new active theme by name. Everything that draws reads
// Active fresh on each render, so the next frame repaints in the new
// palette. Exported so the theme picker modal can apply themes live as the
// cursor moves.
//
// Returns false if name is not in the registry, so the caller can report
// the error. When config file persistence lands, this function gains a
// tea.Cmd return that writes the chosen name to disk - one line added,
// no caller changes needed.
func SetTheme(name string) bool {
	if t, ok := Themes[name]; ok {
		Active = t
		return true
	}
	return false
}
