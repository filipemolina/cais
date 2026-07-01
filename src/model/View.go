package model

import (
	"stack-stitcher/src/appstyles"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m AppModel) View() tea.View {
	mainMenu := m.components.MainMenu.View().Content
	listView := m.components.ServicesList.View().Content
	detailsView := m.components.DetailsPanel.View().Content

	body := lipgloss.JoinHorizontal(lipgloss.Top, listView, detailsView)

	layout := lipgloss.JoinVertical(lipgloss.Left, mainMenu, body)

	v := tea.NewView(appstyles.DocStyle.Render(layout))
	v.AltScreen = true

	return v
}
