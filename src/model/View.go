package model

import (
	"stack-stitcher/src/appstyles"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m AppModel) View() tea.View {
	mainMenu := m.components.MainMenu.View().Content
	// servicesList := m.components.ServicesList.View().Content
	profilesList := m.components.ProfilesList.View().Content
	detailsView := m.components.DetailsPanel.View().Content

	body := lipgloss.JoinHorizontal(lipgloss.Top, profilesList, detailsView)

	layout := lipgloss.JoinVertical(lipgloss.Left, mainMenu, body)

	v := tea.NewView(appstyles.DocStyle.Render(layout))
	v.AltScreen = true

	return v
}
