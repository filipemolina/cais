package model

import (
	"stack-stitcher/src/apptypes"
	"stack-stitcher/src/cmds"

	tea "charm.land/bubbletea/v2"
)

func (m AppModel) Init() tea.Cmd {
	initialCommands := []tea.Cmd{
		cmds.SetActivePage(apptypes.PageTitles[0]),
		cmds.SetFocus(1),
		cmds.GetRunningContainers,
		cmds.GetConfig,
	}

	return tea.Batch(initialCommands...)
}
