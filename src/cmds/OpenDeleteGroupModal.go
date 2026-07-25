package cmds

import tea "charm.land/bubbletea/v2"

type OpenDeleteGroupModalMsg string

func OpenDeleteGroupModal(groupName string) tea.Cmd {
	return func() tea.Msg { return OpenDeleteGroupModalMsg(groupName) }
}
