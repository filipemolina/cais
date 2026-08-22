package cmds

import tea "charm.land/bubbletea/v2"

// ToggleUngroupedRequestMsg asks AppModel to show the adopt/release confirm
// for the reserved ungrouped row. AppModel decides which verb applies from
// the loaded project; the list only knows the row is selected.
type ToggleUngroupedRequestMsg struct{}

// RequestToggleUngrouped asks AppModel to open the confirm for the reserved
// ungrouped row's materialization toggle.
func RequestToggleUngrouped() tea.Cmd {
	return func() tea.Msg {
		return ToggleUngroupedRequestMsg{}
	}
}
