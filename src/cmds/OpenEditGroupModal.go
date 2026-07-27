package cmds

import tea "charm.land/bubbletea/v2"

// OpenEditGroupModalMsg asks AppModel to open the service checklist pre-
// checked with groupName's current members. The modal emits an
// EditGroupRequestMsg on save; AppModel turns it into EditGroup with the
// loaded file.
type OpenEditGroupModalMsg struct {
	GroupName string
}

func OpenEditGroupModal(groupName string) tea.Cmd {
	return func() tea.Msg { return OpenEditGroupModalMsg{GroupName: groupName} }
}
