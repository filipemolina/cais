package mainmenu

import (
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/cmds"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width

	case cmds.SetActivePageMsg:
		if idx := slices.Index(m.items, string(msg)); idx >= 0 {
			m.selectedItemIndex = idx
		}
	}

	return m, nil
}
