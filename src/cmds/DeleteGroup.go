package cmds

import (
	"github.com/filipemolina/stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

type DeleteGroupMsg struct {
	Err error
}

// DeleteGroup removes a group tag from every service that carries it
// in the compose file on disk.
func DeleteGroup(name string) tea.Cmd {
	return func() tea.Msg {
		fileName, err := utils.GetComposeFileName()
		if err != nil {
			return DeleteGroupMsg{Err: err}
		}

		return DeleteGroupMsg{Err: utils.RemoveGroupTag(fileName, name)}
	}
}
