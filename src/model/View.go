package model

import (
	"stack-stitcher/src/appstyles"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m AppModel) View() tea.View {
	var v tea.View
	mainMenu := m.components.MainMenu.View().Content
	pageComponents, ok := m.pages[m.activePage]

	if ok {
		var contents []string

		for idx, _ := range pageComponents {
			contents = append(contents, pageComponents[idx].View().Content)
		}

		body := lipgloss.JoinHorizontal(lipgloss.Top, contents...)

		layout := lipgloss.JoinVertical(lipgloss.Left, mainMenu, body)
		v = tea.NewView(appstyles.DocStyle.Render(layout))

		v.AltScreen = true
	}

	return v
}
