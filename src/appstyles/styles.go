package appstyles

import "charm.land/lipgloss/v2"

// Semantic tokens used across the UI. The single source of truth for color.
// Add new tokens here rather than scattering hex values in components.

// Accent is the primary purple/magenta used for focus, branding, and CTAs.
var Accent = lipgloss.Color("#BC3FBC")

// Text tokens.
var TextPrimary = lipgloss.Color("#FAFAFA")
var TextMuted = lipgloss.Darken(TextPrimary, 0.2)
var TextDim = lipgloss.Darken(TextPrimary, 0.5)

// Panel/surface tokens.
var PanelBg = lipgloss.Color("#151520")
var PanelBgActive = lipgloss.Lighten(PanelBg, 0.05)
var SurfaceBg = lipgloss.Color("#3F3F3F")

// Three-tier background system, layered to separate sections visually
// without relying on borders. See docs/DESIGN.md for the model.
//
//	Tier 1 (terminal default) - outside the app
//	Tier 2 (BackgroundContent) - the "frame": top nav, bottom keybinding bar
//	Tier 3 (BackgroundPanel)   - the "main content": left + right panels
//	Tier 4 (BackgroundElevated) - the "selection": focused panel, modals
//
// The focus state is shown by lifting a panel from tier 3 to tier 4
// (a subtle background lightening), not by a thicker border. Accent color
// is reserved for the nav's active underline and the keybinding bar's key
// labels.
var BackgroundContent = lipgloss.Lighten(PanelBg, 0.04)
var BackgroundPanel = lipgloss.Lighten(PanelBg, 0.08)
var BackgroundElevated = lipgloss.Lighten(PanelBg, 0.12)

// Border tokens.
var BorderDefault = lipgloss.Darken(PanelBg, 0.3)
var BorderFocus = Accent

// Status tokens.
var StatusRunning = lipgloss.Color("#67C58A")
var StatusStopped = lipgloss.Color("#858392")
var StatusStarting = lipgloss.Color("#E8C547")
var StatusError = lipgloss.Color("#EB4268")

// Backward-compatible aliases for existing call sites.
var PrimaryColor = Accent
var PaneColor = PanelBg
var PanelBackgroundColor = SurfaceBg
var SelectedPaneColor = lipgloss.Darken(PrimaryColor, 0.5)
var FocusedPaneColor = lipgloss.Darken(PrimaryColor, 0.7)
var PrimaryFontColor = TextPrimary
var SecondaryFontColor = TextMuted

var lightDark = lipgloss.LightDark(false)

var DocStyle = lipgloss.NewStyle()

var BackgroundColor = lipgloss.Darken(PrimaryColor, 0.5)
var FocusedBackgroundColor = PrimaryColor
var ComplementaryColor = lipgloss.Complementary(PrimaryColor)

var NormalTitle = lipgloss.NewStyle().
	Foreground(TextPrimary).
	Background(PrimaryColor).
	Padding(0, 1).
	MarginLeft(2)

var NormalDesc = NormalTitle.
	Foreground(lightDark(lipgloss.Color("#A49FA5"), lipgloss.Color("#777777")))

var SelectedTitle = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder(), false, false, false, true).
	BorderForeground(lightDark(lipgloss.Color("#F793FF"), lipgloss.Color("#AD58B4"))).
	Foreground(lightDark(lipgloss.Color("#EE6FF8"), lipgloss.Color("#EE6FF8"))).
	Padding(0, 0, 0, 1)

var SelectedDesc = SelectedTitle.
	Foreground(lightDark(lipgloss.Color("#F793FF"), lipgloss.Color("#AD58B4")))

var DimmedTitle = lipgloss.NewStyle().
	Foreground(lightDark(lipgloss.Color("#A49FA5"), lipgloss.Color("#777777"))).
	Padding(0, 0, 0, 2) //nolint:mnd

var DimmedDesc = DimmedTitle.
	Foreground(lightDark(lipgloss.Color("#C2B8C2"), lipgloss.Color("#4D4D4D")))
