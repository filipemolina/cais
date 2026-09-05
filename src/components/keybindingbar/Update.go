package keybindingbar

import (
	tea "charm.land/bubbletea/v2"

	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/keys"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width

	// The whole context arrives at once, already resolved. Everything the bar
	// used to mirror to build it - the page, the selection, three list
	// emptiness flags, the filter state, editing, pending actions, the
	// ungrouped row's backing and the Backups focus - is inside this.
	case cmds.SetKeyContextMsg:
		m.keyContext = keys.Context(msg)

	case cmds.SetComposeFileMsg:
		m.composeFile = msg.Name
		m.composeFileOthers = len(msg.Others)
	}
	return m, nil
}
