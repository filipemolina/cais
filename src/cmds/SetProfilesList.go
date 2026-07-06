package cmds

import tea "charm.land/bubbletea/v2"

type SetProfilesListMsg []string

func SetProfilesLit(profiles []string) tea.Cmd {
	return func() tea.Msg { return SetProfilesListMsg(profiles) }
}
