package cmds

import tea "charm.land/bubbletea/v2"

type SetActivePageMsg int

func SetActivePage(idx int) func() tea.Msg {
	return func() tea.Msg {
		return SetActivePageMsg(idx)
	}
}
