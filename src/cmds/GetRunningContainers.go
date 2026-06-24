package cmds

import (
	"stack-stitcher/src/apptypes"
	"stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

type GetRunningContainersMsg []apptypes.DockerContainer

func GetRunningContainers() tea.Msg {
	commandOutput := utils.DockerPs()
	containers := utils.ParseContainers(commandOutput)

	return GetRunningContainersMsg(containers)
}
