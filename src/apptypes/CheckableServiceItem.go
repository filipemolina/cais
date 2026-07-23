package apptypes

import "fmt"

// CheckableServiceItem is a list.Item for the service-selection checklist
// shown when creating a profile.
type CheckableServiceItem struct {
	Name    string
	Checked bool
}

func (s CheckableServiceItem) Title() string {
	box := "[ ]"
	if s.Checked {
		box = "[x]"
	}

	return fmt.Sprintf("%s %s", box, s.Name)
}

func (s CheckableServiceItem) FilterValue() string { return s.Name }
