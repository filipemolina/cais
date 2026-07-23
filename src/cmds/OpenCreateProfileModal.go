package cmds

import tea "charm.land/bubbletea/v2"

type OpenCreateProfileModalMsg struct{}

func OpenCreateProfileModal() tea.Cmd {
	return func() tea.Msg { return OpenCreateProfileModalMsg{} }
}
