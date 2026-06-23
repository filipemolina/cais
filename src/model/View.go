package model

import (
	tea "charm.land/bubbletea/v2"
)

func (m AppModel) View() tea.View {
	return tea.NewView(m.containers.runningContainers.View())
}
