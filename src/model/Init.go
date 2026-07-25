package model

import (
	"stack-stitcher/src/apptypes"
	"stack-stitcher/src/cmds"
	"stack-stitcher/src/constants"

	tea "charm.land/bubbletea/v2"
)

func (m AppModel) Init() tea.Cmd {
	initialCommands := []tea.Cmd{
		cmds.SetActivePage(apptypes.PageTitles[0]),
		cmds.SetFocus(constants.COMPONENT_BODY_LIST),
		cmds.GetRunningContainers,
		cmds.GetConfig,
	}

	return tea.Batch(initialCommands...)
}
