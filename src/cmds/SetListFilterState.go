package cmds

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

// SetListFilterStateMsg carries how much of the keyboard the focused list has
// taken, so the footer can advertise the keys that actually work: while a filter
// is being typed the panel verbs are inert, and once one is applied there is an
// esc that clears it which nothing else would mention.
//
// It is broadcast on the transition rather than every keystroke, the same as
// every other Set* message here.
type SetListFilterStateMsg list.FilterState

func SetListFilterState(state list.FilterState) tea.Cmd {
	return func() tea.Msg { return SetListFilterStateMsg(state) }
}
