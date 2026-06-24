package cmds

import (
	"stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

type GetConfigMsg = struct {
	FileName string
	Project  *types.Project
}

func GetConfig() tea.Msg {
	fileName := utils.GetComposeFileName()
	project := utils.ReadConfigFile(fileName)

	return GetConfigMsg{
		FileName: fileName,
		Project:  project,
	}
}
