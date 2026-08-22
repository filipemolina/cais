package cmds

import tea "charm.land/bubbletea/v2"

// SetUngroupedMaterializedMsg tells the footer whether the reserved ungrouped
// row is backed by a written profile tag (materialized) or derived, so it can
// advertise the right 'A' verb (adopt vs release).
type SetUngroupedMaterializedMsg bool

// SetUngroupedMaterialized broadcasts the materialized state of the reserved
// ungrouped profile after a config reload.
func SetUngroupedMaterialized(materialized bool) tea.Cmd {
	return func() tea.Msg {
		return SetUngroupedMaterializedMsg(materialized)
	}
}
