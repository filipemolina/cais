package components

import (
	"stack-stitcher/src/appstyles"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type ButtonModel struct {
	text     string
	shortcut string
}

func (m ButtonModel) Init() tea.Cmd {
	return nil
}

func (m ButtonModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m ButtonModel) View() tea.View {
	buttonStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		Padding(0, 1).
		BorderForeground(appstyles.PrimaryColor)

	return tea.NewView(buttonStyle.Render(m.shortcut + " " + m.text))
}

func Button(text string, shortcut string) tea.Model {
	return ButtonModel{
		text:     text,
		shortcut: shortcut,
	}
}
