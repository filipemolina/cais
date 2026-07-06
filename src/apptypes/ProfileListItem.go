package apptypes

type ProfileListItem string

func (s ProfileListItem) Title() string       { return string(s) }
func (s ProfileListItem) FilterValue() string { return string(s) }
