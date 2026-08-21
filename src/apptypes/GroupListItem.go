package apptypes

// GroupListItem is one row in the groups list. It carries the group's name
// plus how many of its member services are running, so the row can render a
// status dot (green when fully running, amber half-circle when mixed).
type GroupListItem struct {
	Name    string
	Running int
	Total   int
}

func (s GroupListItem) Title() string       { return s.Name }
func (s GroupListItem) FilterValue() string { return s.Name }
