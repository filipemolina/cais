package cmds

import tea "charm.land/bubbletea/v2"

// SetEditingStateMsg tells the app whether the service details panel is in
// inline edit mode. The footer needs this to swap the action hints for the
// editor hints, and AppModel uses it to keep the help overlay in sync.
type SetEditingStateMsg bool

// SetEditingState returns a message that broadcasts the panel's edit state.
func SetEditingState(editing bool) tea.Cmd {
	return func() tea.Msg { return SetEditingStateMsg(editing) }
}
