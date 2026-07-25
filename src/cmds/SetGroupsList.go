package cmds

import tea "charm.land/bubbletea/v2"

type SetGroupsListMsg []string

func SetGroupsList(groups []string) tea.Cmd {
	return func() tea.Msg { return SetGroupsListMsg(groups) }
}
