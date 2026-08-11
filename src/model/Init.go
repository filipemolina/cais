package model

import (
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/cmds"

	tea "charm.land/bubbletea/v2"
)

func (m AppModel) Init() tea.Cmd {
	initialCommands := []tea.Cmd{
		cmds.SetActivePage(apptypes.PageTitles[0]),
		cmds.GetConfig(m.config.source),
		cmds.RefreshContainersTick(),
		cmds.CheckDocker(),
	}

	return tea.Batch(initialCommands...)
}
