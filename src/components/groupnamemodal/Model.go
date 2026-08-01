package groupnamemodal

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type Model struct {
	input          textinput.Model
	existingGroups []string
	serviceNames   []string
	errMsg         string
	// isRename switches step 1 of the create flow into the whole rename
	// flow: Enter writes the rename instead of advancing to the service
	// checklist, and uniqueness is judged against the other groups - the
	// name being renamed is not a collision.
	isRename bool
	// currentName is the group being renamed; the input is pre-filled with
	// it and a submit that leaves it unchanged is refused.
	currentName string
	// termHeight is carried rather than used here: this modal is one text
	// input tall, but step 2 is a list of every service and has to be sized
	// to the screen. Update builds it directly, so the height has to arrive
	// with the flow rather than from AppModel at that point.
	termHeight int
}

func (m Model) Init() tea.Cmd {
	return nil
}

// New is step 1 of the create-group flow: prompt for a new, unique group
// name. Enter with a valid name advances to servicechecklistmodal.New; Esc
// cancels the whole flow.
func New(existingGroups []string, serviceNames []string, termHeight int) tea.Model {
	input := textinput.New()
	input.Placeholder = "e.g. core"
	input.SetWidth(30)
	input.Focus()

	return Model{
		input:          input,
		existingGroups: existingGroups,
		serviceNames:   serviceNames,
		termHeight:     termHeight,
	}
}

// NewForRename is the rename flow: prompt for the group's new name,
// pre-filled with the current one (cursor at end; ctrl+u clears it
// wholesale). Enter writes the rename via RequestRenameGroup; Esc cancels.
// Uniqueness excludes the current name, so renaming core to core gets its
// own message rather than "already exists". No termHeight: unlike the
// create flow there is no step-2 checklist to size to the screen.
func NewForRename(currentName string, existingGroups []string) tea.Model {
	input := textinput.New()
	input.Placeholder = "e.g. core"
	input.SetWidth(30)
	input.SetValue(currentName)
	input.Focus()

	return Model{
		input:          input,
		existingGroups: existingGroups,
		isRename:       true,
		currentName:    currentName,
	}
}
