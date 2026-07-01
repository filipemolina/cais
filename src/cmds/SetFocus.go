package cmds

import tea "charm.land/bubbletea/v2"

type SetFocusMsg int

func SetFocus(idx int) func() tea.Msg {
	return func() tea.Msg {
		return SetFocusMsg(idx)
	}
}
