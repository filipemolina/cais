package cmds

import (
	"github.com/filipemolina/cais/src/utils"

	tea "charm.land/bubbletea/v2"
)

type CreateGroupMsg struct {
	Err error
}

// CreateGroupRequestMsg asks AppModel to create a group. The checklist modal
// emits this instead of the command itself: like the details panels, it has
// no business knowing which compose file is loaded.
type CreateGroupRequestMsg struct {
	Name     string
	Services []string
}

// RequestCreateGroup asks AppModel to tag serviceNames with the group name.
func RequestCreateGroup(name string, serviceNames []string) tea.Cmd {
	return func() tea.Msg {
		return CreateGroupRequestMsg{Name: name, Services: serviceNames}
	}
}

// CreateGroup tags each of the given services with a new group name in
// fileName, the compose file AppModel has loaded.
func CreateGroup(fileName string, name string, serviceNames []string) tea.Cmd {
	return func() tea.Msg {
		return CreateGroupMsg{Err: utils.AddGroupTag(fileName, name, serviceNames)}
	}
}
