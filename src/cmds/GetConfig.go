package cmds

import (
	"stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

type GetConfigMsg = struct {
	FileName string
	Project  *types.Project
	Err      error
}

func GetConfig() tea.Msg {
	fileName, err := utils.GetComposeFileName()
	if err != nil {
		return GetConfigMsg{Err: err}
	}

	project, err := utils.ReadConfigFile(fileName)
	if err != nil {
		return GetConfigMsg{Err: err}
	}

	return GetConfigMsg{
		FileName: fileName,
		Project:  project,
	}
}
