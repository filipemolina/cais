package cmds

import (
	"stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

type CreateGroupMsg struct {
	Err error
}

// CreateGroup tags each of the given services with a new group name in
// the compose file on disk.
func CreateGroup(name string, serviceNames []string) tea.Cmd {
	return func() tea.Msg {
		fileName, err := utils.GetComposeFileName()
		if err != nil {
			return CreateGroupMsg{Err: err}
		}

		return CreateGroupMsg{Err: utils.AddGroupTag(fileName, name, serviceNames)}
	}
}
