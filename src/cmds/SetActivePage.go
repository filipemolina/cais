package cmds

import tea "charm.land/bubbletea/v2"

type SetActivePageMsg string

func SetActivePage(pageTitle string) func() tea.Msg {
	return func() tea.Msg {
		return SetActivePageMsg(pageTitle)
	}
}
