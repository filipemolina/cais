package cmds

import tea "charm.land/bubbletea/v2"

// GroupStatus carries one group's name and how many of its member services
// are running, so the groups list can render a status dot per row.
type GroupStatus struct {
	Name    string
	Running int
	Total   int
	// Stale is true when any member service's image check came back with an
	// update available. It rides the status broadcast because only AppModel
	// knows which services belong to which group; the row cannot derive it
	// from the update map alone.
	Stale bool
}

type SetGroupsListMsg []GroupStatus

func SetGroupsList(groups []GroupStatus) tea.Cmd {
	return func() tea.Msg { return SetGroupsListMsg(groups) }
}
