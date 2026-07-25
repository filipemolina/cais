package cmds

import tea "charm.land/bubbletea/v2"

type OpenCreateGroupModalMsg struct{}

func OpenCreateGroupModal() tea.Cmd {
	return func() tea.Msg { return OpenCreateGroupModalMsg{} }
}
