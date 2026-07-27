package cmds

import (
	"github.com/filipemolina/stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

type GetConfigMsg = struct {
	FileName string
	// Files is every compose-file candidate that exists in the directory, in
	// Docker's priority order, so Files[0] is FileName. The rest are the
	// candidates that lost - the footer counts them and the help overlay
	// lists them. Empty in tests that construct the message by hand.
	Files   []string
	Project *types.Project
	Err     error
}

func GetConfig() tea.Msg {
	fileName, candidates, err := utils.GetComposeFileName()
	if err != nil {
		return GetConfigMsg{Err: err}
	}

	project, err := utils.ReadConfigFile(fileName)
	if err != nil {
		return GetConfigMsg{Err: err}
	}

	return GetConfigMsg{
		FileName: fileName,
		Files:    candidates,
		Project:  project,
	}
}
