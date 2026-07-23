package cmds

import (
	"stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

type DeleteProfileMsg struct {
	Err error
}

// DeleteProfile removes a profile tag from every service that carries it
// in the compose file on disk.
func DeleteProfile(name string) tea.Cmd {
	return func() tea.Msg {
		fileName, err := utils.GetComposeFileName()
		if err != nil {
			return DeleteProfileMsg{Err: err}
		}

		return DeleteProfileMsg{Err: utils.RemoveProfileTag(fileName, name)}
	}
}
