package cmds

import (
	"stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

func GetRunningContainers() tea.Msg {
	commandOutput := utils.DockerPs()
	containers := utils.ParseContainers(commandOutput)

	return containers
}
