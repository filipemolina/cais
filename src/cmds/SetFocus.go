package cmds

import tea "charm.land/bubbletea/v2"

type SetFocusMsg int

func SetFocus(idx int) tea.Msg {
	return SetFocusMsg(idx)
}
