package cmds

import (
	"stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

type CreateProfileMsg struct {
	Err error
}

// CreateProfile tags each of the given services with a new profile name in
// the compose file on disk.
func CreateProfile(name string, serviceNames []string) tea.Cmd {
	return func() tea.Msg {
		fileName, err := utils.GetComposeFileName()
		if err != nil {
			return CreateProfileMsg{Err: err}
		}

		return CreateProfileMsg{Err: utils.AddProfileTag(fileName, name, serviceNames)}
	}
}
