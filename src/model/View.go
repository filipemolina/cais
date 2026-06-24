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

	return tea.NewView("")
}
