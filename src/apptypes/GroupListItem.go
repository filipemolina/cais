package apptypes

type GroupListItem string

func (s GroupListItem) Title() string       { return string(s) }
func (s GroupListItem) FilterValue() string { return string(s) }
