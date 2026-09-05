package chrome

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/filipemolina/cais/src/appstyles"
)

var WrapperStyle = lipgloss.NewStyle().
	Padding(1, 2)

// ListWrapperStyle is the frame around the two body lists. Its padding is
// what separates the list content from the panel edges, and its frame size is
// subtracted from the panel box when the inner list is sized.
var ListWrapperStyle = lipgloss.NewStyle().
	Padding(1, 2, 2, 2)

// FitBox constrains a style to an exact w x h box: Width/Height pad it out,
// Max* clip anything that would otherwise overflow (Width alone pads but
// never truncates, which is how a too-wide panel ends up wrapped by the
// terminal). Non-positive dimensions are left unset so a component still
// renders naturally before the first SetBodyLayoutMsg arrives.
func FitBox(s lipgloss.Style, w, h int) lipgloss.Style {
	if w > 0 {
		s = s.Width(w).MaxWidth(w)
	}

	if h > 0 {
		s = s.Height(h).MaxHeight(h)
	}

	return s
}

// PanelBg is the background tier a body panel renders on. Both body panels are
// always active now that focus is gone, so they share the elevated tier (what
// the previously "focused" panel used to get). Focus used to lift the whole
// panel rather than adding a border, so the panel's box stayed the same size
// either way - that lift is now the steady state.
func PanelBg() color.Color {
	return appstyles.Active.BackgroundElevated
}

// PanelBgFor is PanelBg for the two panels that still have focus to show: the
// Backups list and its preview. The focused one sits on the elevated tier, the
// unfocused one a tier below, which is the lift DESIGN.md documents - focus
// changes the tier, never the border, so the box stays the same size either
// way.
//
// It is a second function rather than a parameter on PanelBg because every
// other panel in the app is always active and has no answer to give: adding
// the argument there would mean threading a constant `true` through a dozen
// call sites to say nothing.
func PanelBgFor(isFocused bool) color.Color {
	if isFocused {
		return appstyles.Active.BackgroundElevated
	}

	return appstyles.Active.BackgroundPanel
}

// ListRowBg is the background a list row renders on. The active row is lifted
// to the surface tier; every other row sits flush on the panel's elevated
// tier. Rows need an explicit background (rather than inheriting the panel's)
// because each row is rendered and sealed on its own - see
// appstyles.FillBackground.
func ListRowBg(isActive bool) color.Color {
	return ListRowBgOn(isActive, PanelBg())
}

// ListRowBgOn is ListRowBg for a list inside a panel whose tier moves: the
// inactive rows have to sit flush on whatever tier the panel is actually on,
// or an unfocused panel shows its rows floating a tier above their own frame.
// It is DESIGN.md's rule that a component inside a panel takes that panel's
// tier as a parameter instead of picking a tint of its own.
//
// The active row's ModalBg is unchanged by focus: it is its own register
// rather than a tint derived from the panel tiers, so it reads as the cursor
// on either tier.
func ListRowBgOn(isActive bool, panelBg color.Color) color.Color {
	if isActive {
		return appstyles.Active.ModalBg
	}

	return panelBg
}

// BarColumn renders the nav's ▌ indicator once per line of content, so the
// bar spans a multi-line row's full height instead of a sliver at its top.
// bg may be nil to leave the cell background unset.
func BarColumn(fg color.Color, bg color.Color, content string) string {
	style := lipgloss.NewStyle().Foreground(fg)
	if bg != nil {
		style = style.Background(bg)
	}

	lines := max(1, strings.Count(content, "\n")+1)
	bar := style.Render("▌")
	return strings.Repeat(bar+"\n", lines-1) + bar
}

// StatusPill renders a filled, bold pill carrying label on the fill color bg.
// Callers choose the label and the fill; this decides the ink and the shape,
// which is the part all three pills were repeating.
//
// The pill's ink (fg) does not follow the app's theme: InkOnLight/InkOnDark
// are fixed regardless of Dark, because the pill's own fill is a status color
// rather than a surface tier. What the fill *is* varies per theme, though, so
// appstyles.InkOn picks whichever of the two inks reads on it rather than the
// call site guessing - see appstyles/Contrast_test.go, which holds every
// theme's pill ink to 4.2:1.
func StatusPill(label string, bg color.Color) string {
	return lipgloss.NewStyle().
		Background(bg).
		Foreground(appstyles.InkOn(bg)).
		Bold(true).
		Padding(0, 1).
		Render(label)
}
