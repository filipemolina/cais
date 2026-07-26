package cmds

import (
	"github.com/filipemolina/stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

type CreateComposeFileMsg struct {
	Err error
}

// CreateComposeFile writes a brand-new compose file at fileName in the
// current working directory, optionally pre-seeded with one service. Empty
// serviceName/image means "create an empty services: mapping".
func CreateComposeFile(fileName string, serviceName string, image string) tea.Cmd {
	return func() tea.Msg {
		return CreateComposeFileMsg{Err: utils.WriteNewComposeFile(fileName, serviceName, image)}
	}
}
