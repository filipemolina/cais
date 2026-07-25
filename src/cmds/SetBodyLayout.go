package cmds

import tea "charm.land/bubbletea/v2"

// SetBodyLayoutMsg is the exact box each body panel must render into.
// AppModel is the single source of truth for these numbers: it guarantees
// LeftWidth + constants.BODY_GUTTER_WIDTH + RightWidth == the terminal
// width, and that Height is the row count left after the nav, the keybinding
// bar, and the optional error banner.
//
// Components must render at exactly this size rather than deriving their own
// from tea.WindowSizeMsg. WindowSizeMsg only reaches the components of the
// page that is active when it arrives, so any page that wasn't active at
// resize time would otherwise render at width 0. Both axes travel in one
// message so they can never be out of sync mid-frame.
type SetBodyLayoutMsg struct {
	LeftWidth  int
	RightWidth int
	Height     int
}

func SetBodyLayout(leftWidth, rightWidth, height int) tea.Cmd {
	return func() tea.Msg {
		return SetBodyLayoutMsg{
			LeftWidth:  leftWidth,
			RightWidth: rightWidth,
			Height:     height,
		}
	}
}
