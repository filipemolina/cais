package model

import (
	tea "charm.land/bubbletea/v2"
)

func (m AppModel) View() tea.View {
	// m.containers.runningContainers.SetShowHelp(false)

	// mainMenu := appstyles.DocStyle.Render(
	// 	m.containers.runningContainers.View(),
	// )

	// content := appstyles.DocStyle.Render("TESTE TESTE TESTE")

	// v := tea.NewView(lipgloss.JoinHorizontal(lipgloss.Top, mainMenu, content))
	// v.AltScreen = true
	s := "Services: \n\n"

	if m.config.configProject != nil {
		for _, service := range m.config.configProject.Services {
			s += service.Name + "\n"
		}
	}

	return tea.NewView(s)
}
