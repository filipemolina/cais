package cmds

import tea "charm.land/bubbletea/v2"

type SetSelectedGroupMsg string

func SetSelectedGroup(group string) tea.Cmd {
	return func() tea.Msg { return SetSelectedGroupMsg(group) }
}
