package model

import tea "charm.land/bubbletea/v2"

func (m AppModel) View() tea.View {
	s := "List of running containers: \n\n"

	if len(m.containers.runningContainers) > 0 {
		for _, container := range m.containers.runningContainers {
			s += container.Names + "\n"
		}
	}

	return tea.NewView(s)
}
