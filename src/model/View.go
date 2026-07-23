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
		rendered := appstyles.DocStyle.Render(layout)

		if m.activeModal != nil {
			rendered = m.renderWithModal(rendered)
		}

		v = tea.NewView(rendered)
		v.AltScreen = true
	}

	return v
}

// renderWithModal composites the active modal as a centered layer on top
// of the rest of the screen.
func (m AppModel) renderWithModal(base string) string {
	modalContent := m.activeModal.View().Content

	x := max(0, (m.config.terminalWidht-lipgloss.Width(modalContent))/2)
	y := max(0, (m.config.terminalHeight-lipgloss.Height(modalContent))/2)

	baseLayer := lipgloss.NewLayer(base)
	modalLayer := lipgloss.NewLayer(modalContent).X(x).Y(y).Z(1)

	return lipgloss.NewCompositor(baseLayer, modalLayer).Render()
}
