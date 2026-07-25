package components

import (
	"stack-stitcher/src/appstyles"

	"charm.land/lipgloss/v2"
)

var wrapperStyle = lipgloss.NewStyle().
	Padding(1, 2)

// listWrapperStyle is the frame around the two body lists. Its padding is
// what separates the list content from the panel edges, and its frame size is
// subtracted from the panel box when the inner list is sized.
var listWrapperStyle = lipgloss.NewStyle().
	Padding(1, 2, 2, 2)

// fitBox constrains a style to an exact w x h box: Width/Height pad it out,
// Max* clip anything that would otherwise overflow (Width alone pads but
// never truncates, which is how a too-wide panel ends up wrapped by the
// terminal). Non-positive dimensions are left unset so a component still
// renders naturally before the first SetBodyLayoutMsg arrives.
func fitBox(s lipgloss.Style, w, h int) lipgloss.Style {
	if w > 0 {
		s = s.Width(w).MaxWidth(w)
	}

	if h > 0 {
		s = s.Height(h).MaxHeight(h)
	}

	return s
}

var logoStyle = lipgloss.NewStyle().
	Align(lipgloss.Center).
	Foreground(appstyles.TextPrimary)
