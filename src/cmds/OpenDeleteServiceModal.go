package cmds

import tea "charm.land/bubbletea/v2"

type OpenDeleteServiceModalMsg string

func OpenDeleteServiceModal(serviceName string) tea.Cmd {
	return func() tea.Msg { return OpenDeleteServiceModalMsg(serviceName) }
}
