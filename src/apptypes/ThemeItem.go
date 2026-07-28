package apptypes

// ThemeItem is a list.Item for the theme picker modal. Active marks the
// theme that was in effect when the modal opened, so the user can see
// which one they started from and Esc can restore it.
type ThemeItem struct {
	Name   string
	Active bool
}

func (t ThemeItem) Title() string {
	if t.Active {
		return t.Name + "  (active)"
	}
	return t.Name
}

func (t ThemeItem) FilterValue() string { return t.Name }
