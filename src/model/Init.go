package model

import (
	"stack-stitcher/src/apptypes"
	"stack-stitcher/src/cmds"

	tea "charm.land/bubbletea/v2"
)

func (m AppModel) Init() tea.Cmd {
	initialCommands := []tea.Cmd{
		cmds.SetActivePage(apptypes.PageTitles[0]),
		cmds.GetConfig,
		cmds.RefreshContainersTick(),
	}

	return tea.Batch(initialCommands...)
}
