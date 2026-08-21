package cmds

import tea "charm.land/bubbletea/v2"

// GroupStatus carries one group's name and how many of its member services
// are running, so the groups list can render a status dot per row.
type GroupStatus struct {
	Name    string
	Running int
	Total   int
}

type SetGroupsListMsg []GroupStatus

func SetGroupsList(groups []GroupStatus) tea.Cmd {
	return func() tea.Msg { return SetGroupsListMsg(groups) }
}
