package appstyles

import (
	"image/color"
	"math"

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
	if Contrast(Active.InkOnLight, fill) >= Contrast(Active.InkOnDark, fill) {
		return Active.InkOnLight
	}
	return Active.InkOnDark
}

// Contrast is the WCAG 2.x contrast ratio between two opaque colors.
func Contrast(a, b color.Color) float64 {
	la := relativeLuminance(a)
	lb := relativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// relativeLuminance is WCAG 2.x relative luminance.
func relativeLuminance(c color.Color) float64 {
	r, g, b := extractRGB(c)
	r = srgbLinearize(r)
	g = srgbLinearize(g)
	b = srgbLinearize(b)
	return 0.2126*r + 0.7152*g + 0.0722*b
}

func extractRGB(c color.Color) (float64, float64, float64) {
	rgba := color.RGBAModel.Convert(c).(color.RGBA)
	return float64(rgba.R) / 255.0, float64(rgba.G) / 255.0, float64(rgba.B) / 255.0
}

func srgbLinearize(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

// DefaultTheme is the theme a fresh AppModel starts with. Cais-dusk is
// the app's default palette (changed 2026-08-18, was cais-dark); cais-dark
// and cais-day remain selectable through the theme picker (docs/DESIGN.md
// §11).
const DefaultTheme = "cais-dusk"

// Themes is the registry a theme picker (post-alpha, see docs/ROADMAP.md)
// will choose from. Every entry is built through newTheme rather than a bare
// struct literal, so a registered theme can't leave a field zero-valued the
// way a hand-written literal could - which matters here because a nil
// color.Color renders as no SGR at all, i.e. a background-bleed bug. See
// src/appstyles/Theme_test.go and src/appstyles/Background_test.go.
var Themes = map[string]Theme{
	// cais-dark — deep-water panel with a harbor-teal accent lifted to
	// clear WCAG 3.0 against the dark surface. See
	// docs/plans/cais/cais-themes.md for the palette rationale.
	"cais-dark": newTheme(themeParams{
		Name:   "cais-dark",
		Dark:   true,
		Accent: lipgloss.Color("#5FB8AF"),
		Text:   lipgloss.Color("#F2EFE9"),
		Panel:  lipgloss.Color("#0F2B33"),
		Modal:  lipgloss.Color("#3A5860"),
		Danger: lipgloss.Color("#E6736E"),

		Running:  lipgloss.Color("#7DC79E"),
		Stopped:  lipgloss.Color("#78939A"),
		Starting: lipgloss.Color("#F5C98A"),
		Err:      lipgloss.Color("#EE7090"),
	}),

	// cais-dusk — default (changed 2026-08-18, was cais-dark). Cool
	// slate-blue panel (NOT teal — a panel pulled from the same teal
	// family would mash the brand color into the background) with the
	// harbor-teal accent one step warmer than cais-dark. Modal lifts ~25
	// channel units above the elevated tier so the channel-diff floor of
	// 14 is met.
	"cais-dusk": newTheme(themeParams{
		Name:   "cais-dusk",
		Dark:   true,
		Accent: lipgloss.Color("#3FA9A0"),
		Text:   lipgloss.Color("#E8E4DC"),
		Panel:  lipgloss.Color("#1A2230"),
		Modal:  lipgloss.Color("#33425E"),
		Danger: lipgloss.Color("#E07060"),

		Running:  lipgloss.Color("#7DC79E"),
		Stopped:  lipgloss.Color("#7E8A9A"),
		Starting: lipgloss.Color("#F2C674"),
		Err:      lipgloss.Color("#EE7090"),
	}),

	// cais-day — Cais inverted onto warm cream, the same identity as
	// cais-dark but flipped. Accent darkened from #5FB8AF so it stays a
	// chip rather than glow on cream, and the status colors darkened to
	// match (a #7DC79E green that reads on a near-black panel washes
	// out on cream).
	"cais-day": newTheme(themeParams{
		Name:   "cais-day",
		Dark:   false,
		Accent: lipgloss.Color("#1F5E5E"),
		Text:   lipgloss.Color("#1B3232"),
		Panel:  lipgloss.Color("#F1ECE2"),
		Modal:  lipgloss.Color("#FBF7EF"),
		Danger: lipgloss.Color("#B33A3A"),

		Running:  lipgloss.Color("#1E7F4E"),
		Stopped:  lipgloss.Color("#5A6F75"),
		Starting: lipgloss.Color("#B07933"),
		Err:      lipgloss.Color("#C0243F"),
	}),

	// catppuccin-mocha — Catppuccin Mocha (github.com/catppuccin/catppuccin)
	"catppuccin-mocha": newTheme(themeParams{
		Name:   "catppuccin-mocha",
		Dark:   true,
		Accent: lipgloss.Color("#CBA6F7"),
		Text:   lipgloss.Color("#CDD6F4"),
		Panel:  lipgloss.Color("#11111B"),
		Modal:  lipgloss.Color("#45475A"),
		Danger: lipgloss.Color("#EBA0AC"),

		Running:  lipgloss.Color("#A6E3A1"),
		Stopped:  lipgloss.Color("#7F849C"),
		Starting: lipgloss.Color("#F9E2AF"),
		Err:      lipgloss.Color("#F38BA8"),
	}),

	// gruvbox-dark — Gruvbox dark (github.com/morhetz/gruvbox)
	"gruvbox-dark": newTheme(themeParams{
		Name:   "gruvbox-dark",
		Dark:   true,
		Accent: lipgloss.Color("#FE8019"),
		Text:   lipgloss.Color("#EBDBB2"),
		Panel:  lipgloss.Color("#1D2021"),
		Modal:  lipgloss.Color("#504945"),
		Danger: lipgloss.Color("#FB4934"),

		Running:  lipgloss.Color("#B8BB26"),
		Stopped:  lipgloss.Color("#928374"),
		Starting: lipgloss.Color("#FABD2F"),
		Err:      lipgloss.Color("#FB4934"),
	}),

	// tokyo-night — Tokyo Night (github.com/folke/tokyonight.nvim)
	"tokyo-night": newTheme(themeParams{
		Name:   "tokyo-night",
		Dark:   true,
		Accent: lipgloss.Color("#7AA2F7"),
		Text:   lipgloss.Color("#C0CAF5"),
		Panel:  lipgloss.Color("#16161E"),
		Modal:  lipgloss.Color("#394B70"),
		Danger: lipgloss.Color("#DB4B4B"),

		Running:  lipgloss.Color("#9ECE6A"),
		Stopped:  lipgloss.Color("#737AA2"),
		Starting: lipgloss.Color("#E0AF68"),
		Err:      lipgloss.Color("#F7768E"),
	}),

	// nord — Nord (nordtheme.com)
	"nord": newTheme(themeParams{
		Name:   "nord",
		Dark:   true,
		Accent: lipgloss.Color("#88C0D0"),
		Text:   lipgloss.Color("#D8DEE9"),
		Panel:  lipgloss.Color("#242933"),
		Modal:  lipgloss.Color("#4C566A"),
		Danger: lipgloss.Color("#BF616A"),

		Running:  lipgloss.Color("#A3BE8C"),
		Stopped:  lipgloss.Color("#7B88A1"),
		Starting: lipgloss.Color("#EBCB8B"),
		Err:      lipgloss.Color("#BF616A"),
	}),

	// dracula — Dracula (draculatheme.com)
	"dracula": newTheme(themeParams{
		Name:   "dracula",
		Dark:   true,
		Accent: lipgloss.Color("#BD93F9"),
		Text:   lipgloss.Color("#F8F8F2"),
		Panel:  lipgloss.Color("#1E1F29"),
		Modal:  lipgloss.Color("#44475A"),
		Danger: lipgloss.Color("#FF5555"),

		Running:  lipgloss.Color("#50FA7B"),
		Stopped:  lipgloss.Color("#6272A4"),
		Starting: lipgloss.Color("#FFB86C"),
		Err:      lipgloss.Color("#FF5555"),
	}),

	// solarized-dark — Solarized Dark (ethanschoonover.com/solarized)
	"solarized-dark": newTheme(themeParams{
		Name:   "solarized-dark",
		Dark:   true,
		Accent: lipgloss.Color("#268BD2"),
		Text:   lipgloss.Color("#93A1A1"),
		Panel:  lipgloss.Color("#001A21"),
		Modal:  lipgloss.Color("#073642"),
		Danger: lipgloss.Color("#DC322F"),

		Running:  lipgloss.Color("#859900"),
		Stopped:  lipgloss.Color("#657B83"),
		Starting: lipgloss.Color("#B58900"),
		Err:      lipgloss.Color("#DC322F"),
	}),

	// everforest-dark — Everforest Dark (github.com/sainnhe/everforest)
	"everforest-dark": newTheme(themeParams{
		Name:   "everforest-dark",
		Dark:   true,
		Accent: lipgloss.Color("#A7C080"),
		Text:   lipgloss.Color("#D3C6AA"),
		Panel:  lipgloss.Color("#232A2E"),
		Modal:  lipgloss.Color("#4F585E"),
		Danger: lipgloss.Color("#E67E80"),

		Running:  lipgloss.Color("#83C092"),
		Stopped:  lipgloss.Color("#859289"),
		Starting: lipgloss.Color("#DBBC7F"),
		Err:      lipgloss.Color("#E67E80"),
	}),

	// rose-pine — Rosé Pine (rosepinetheme.com). Accent is Rose, not Iris -
	// Iris duplicated catppuccin-mocha/dracula's purple, and Rose is unused
	// elsewhere in this theme (Foam and Gold are already Running/Starting).
	"rose-pine": newTheme(themeParams{
		Name:   "rose-pine",
		Dark:   true,
		Accent: lipgloss.Color("#EBBCBA"),
		Text:   lipgloss.Color("#E0DEF4"),
		Panel:  lipgloss.Color("#191724"),
		Modal:  lipgloss.Color("#403D52"),
		Danger: lipgloss.Color("#EB6F92"),

		Running:  lipgloss.Color("#9CCFD8"),
		Stopped:  lipgloss.Color("#908CAA"),
		Starting: lipgloss.Color("#F6C177"),
		Err:      lipgloss.Color("#EB6F92"),
	}),

	// monokai-pro — Monokai Pro (monokai.pro). Danger uses classic Monokai's
	// red-magenta (#F92672), distinct from Pro's coral-pink accent.
	"monokai-pro": newTheme(themeParams{
		Name:   "monokai-pro",
		Dark:   true,
		Accent: lipgloss.Color("#FF6188"),
		Text:   lipgloss.Color("#FCFCFA"),
		Panel:  lipgloss.Color("#221F22"),
		Modal:  lipgloss.Color("#2E272E"),
		Danger: lipgloss.Color("#F92672"),

		Running:  lipgloss.Color("#A9DC76"),
		Stopped:  lipgloss.Color("#7C797D"),
		Starting: lipgloss.Color("#FFD866"),
		Err:      lipgloss.Color("#F92672"),
	}),

	// gruvbox-light — Gruvbox Light (github.com/morhetz/gruvbox). Accent is
	// the neutral red, not the orange gruvbox-dark already owns.
	"gruvbox-light": newTheme(themeParams{
		Name:   "gruvbox-light",
		Dark:   false,
		Accent: lipgloss.Color("#CC241D"),
		Text:   lipgloss.Color("#3C3836"),
		Panel:  lipgloss.Color("#F0EBC8"),
		Modal:  lipgloss.Color("#FFFDF0"),
		Danger: lipgloss.Color("#9D0006"),

		Running:  lipgloss.Color("#79740E"),
		Stopped:  lipgloss.Color("#7C6F64"),
		Starting: lipgloss.Color("#B57614"),
		Err:      lipgloss.Color("#9D0006"),
	}),

	// catppuccin-latte — Catppuccin Latte (github.com/catppuccin/catppuccin).
	// Text and Accent are darkened from Catppuccin's own Text/Yellow swatches
	// - those are tuned for contrast against Latte's Base, but TextMuted/
	// TextDim lighten further still, and Panel here is a derived tier, not
	// Base directly.
	"catppuccin-latte": newTheme(themeParams{
		Name:   "catppuccin-latte",
		Dark:   false,
		Accent: lipgloss.Color("#9A6300"),
		Text:   lipgloss.Color("#33364A"),
		Panel:  lipgloss.Color("#E9EBF0"),
		Modal:  lipgloss.Color("#F8F9FC"),
		Danger: lipgloss.Color("#D20F39"),

		Running:  lipgloss.Color("#2B7A1C"),
		Stopped:  lipgloss.Color("#6B6F80"),
		Starting: lipgloss.Color("#B24C08"),
		Err:      lipgloss.Color("#D20F39"),
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
