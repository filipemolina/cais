package appstyles

import "charm.land/lipgloss/v2"

// Computed styles built from the active theme. A raw color is a Theme field
// (see Theme.go) and is read directly - appstyles.Active.TextPrimary, say.
// Anything that needs actual style logic - more than reading one field -
// lives here as a function instead of a package-level var, so it re-reads
// Active on every call rather than freezing whichever theme was active at
// package init. NormalTitle used to be exactly that kind of frozen var; the
// comment stayed to explain why it no longer is.

// DocStyle is an empty style kept only for its frame size (Padding/Border),
// which does not depend on color - see ContainersList.go's WindowSizeMsg
// handling.
var DocStyle = lipgloss.NewStyle()

// NormalTitle is the title chip on a panel frame - see PanelFrame.go.
func NormalTitle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(InkOn(Active.Accent)).
		Background(Active.Accent).
		Padding(0, 1).
		MarginLeft(2)
}
