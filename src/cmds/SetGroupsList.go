package cmds

import tea "charm.land/bubbletea/v2"

type SetProfilesListMsg []string

func SetProfilesList(profiles []string) tea.Cmd {
	return func() tea.Msg { return SetProfilesListMsg(profiles) }
}
