package model

import (
	"stack-stitcher/src/appstyles"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var errorBannerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FAFAFA")).
	Background(lipgloss.Color("#B33A3A")).
	Padding(0, 1)

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

		sections := []string{mainMenu, body}
		if m.lastError != "" {
			sections = []string{errorBannerStyle.Render("Error: " + m.lastError), mainMenu, body}
		}

		layout := lipgloss.JoinVertical(lipgloss.Left, sections...)
		v = tea.NewView(appstyles.DocStyle.Render(layout))

		v.AltScreen = true
	}

	return v
}
