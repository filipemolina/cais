package cmds

import (
	"stack-stitcher/src/apptypes"
	"stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

type GetRunningContainersMsg struct {
	Containers []apptypes.DockerContainer
	Err        error
	// Background marks a result produced by the periodic poll rather than by
	// a page switch or a docker action. AppModel uses it to keep background
	// results from clearing an error banner they didn't cause.
	Background bool
}

// GetRunningContainers is the foreground refresh: on success it clears the
// error banner, on failure it sets it.
func GetRunningContainers() tea.Msg {
	return getRunningContainers(false)
}

// GetRunningContainersBackground is the ticker-driven poll: it refreshes
// container state and surfaces new docker errors, but a success leaves an
// error banner from another source (e.g. a failed action) alone.
func GetRunningContainersBackground() tea.Msg {
	return getRunningContainers(true)
}

func getRunningContainers(background bool) tea.Msg {
	commandOutput, err := utils.DockerComposePs()
	if err != nil {
		return GetRunningContainersMsg{Err: err, Background: background}
	}

	containers, err := utils.ParseContainers(commandOutput)
	if err != nil {
		return GetRunningContainersMsg{Err: err, Background: background}
	}

	return GetRunningContainersMsg{Containers: containers, Background: background}
}
