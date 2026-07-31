package cmds

import tea "charm.land/bubbletea/v2"

// OpenRenameGroupModalMsg asks AppModel to open the rename prompt for
// groupName. The modal emits a RenameGroupRequestMsg on save; AppModel
// turns it into RenameGroup with the loaded file.
type OpenRenameGroupModalMsg struct {
	GroupName string
}

func OpenRenameGroupModal(groupName string) tea.Cmd {
	return func() tea.Msg { return OpenRenameGroupModalMsg{GroupName: groupName} }
}
