package apptypes

// ComposeFileItem is a list.Item for the file picker on the Files page.
// Active marks the file currently loaded, so the picker can say which one
// the user is already looking at.
type ComposeFileItem struct {
	Name   string
	Active bool
}

func (s ComposeFileItem) Title() string {
	if s.Active {
		return s.Name + "  (active)"
	}

	return s.Name
}

func (s ComposeFileItem) FilterValue() string { return s.Name }
