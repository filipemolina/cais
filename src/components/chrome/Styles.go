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

// ListRowBg is the background a list row renders on. The active row is lifted
// to the surface tier; every other row sits flush on the panel's elevated
// tier. Rows need an explicit background (rather than inheriting the panel's)
// because each row is rendered and sealed on its own - see
// appstyles.FillBackground.
func ListRowBg(isActive bool) color.Color {
	if isActive {
		return appstyles.Active.ModalBg
	}

	return PanelBg()
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
