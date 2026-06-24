package cmds

import (
	"stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

func GetConfig() tea.Msg {
	fileName := utils.GetComposeFileName()

	return fileName
}
