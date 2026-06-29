package model

import (
	"stack-stitcher/src/appstyles"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m AppModel) View() tea.View {
	mainMenu := appstyles.DocStyle.Render(
		m.components.MainMenu.View().Content,
	)

	listView := appstyles.DocStyle.Render(m.components.ServicesList.View().Content)

	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, mainMenu, listView))
	v.AltScreen = true

	return v
}
