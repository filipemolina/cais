package cmds

import tea "charm.land/bubbletea/v2"

type SetSelectedProfileMsg string

func SetSelectedProfile(profile string) tea.Cmd {
	return func() tea.Msg { return SetSelectedProfile(profile) }
}
