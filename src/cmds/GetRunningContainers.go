package cmds

import (
	"stack-stitcher/src/apptypes"
	"stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

type GetRunningContainersMsg struct {
	Containers []apptypes.DockerContainer
	Err        error
}

func GetRunningContainers() tea.Msg {
	commandOutput, err := utils.DockerComposePs()
	if err != nil {
		return GetRunningContainersMsg{Err: err}
	}

	containers, err := utils.ParseContainers(commandOutput)
	if err != nil {
		return GetRunningContainersMsg{Err: err}
	}

	return GetRunningContainersMsg{Containers: containers}
}
