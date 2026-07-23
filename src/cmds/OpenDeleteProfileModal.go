package cmds

import tea "charm.land/bubbletea/v2"

type OpenDeleteProfileModalMsg string

func OpenDeleteProfileModal(profileName string) tea.Cmd {
	return func() tea.Msg { return OpenDeleteProfileModalMsg(profileName) }
}
