package components

import (
	"image/color"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
)

type ButtonModel struct {
	text     string
	shortcut string
	// bg is the background the button is drawn on. A button carries its
	// parent's tier rather than a tint of its own, so it stays flush with the
	// panel when focus lifts that panel from tier 3 to tier 4. Without an
	// explicit background the label would be the one run on the line with no
	// background set, showing the terminal's color through the button.
	bg color.Color
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
		BorderForeground(appstyles.PrimaryColor).
		Background(m.bg).
		BorderBackground(m.bg)

	return tea.NewView(buttonStyle.Render(m.shortcut + " " + m.text))
}

func Button(text string, shortcut string, bg color.Color) tea.Model {
	return ButtonModel{
		text:     text,
		shortcut: shortcut,
		bg:       bg,
	}
}
