package envkeymodal

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
)

func (m Model) View() tea.View {
	sections := []string{
		chrome.ModalTitle("Add Variable"),
		"Variable Name:",
		m.keyInput.View(),
	}

	if m.err != nil {
		sections = append(sections, lipgloss.NewStyle().
			Foreground(appstyles.Active.Danger).
			Render("Error: "+m.err.Error()))
	}

	content := strings.Join(sections, "\n")
	return tea.NewView(chrome.ModalSurface(appstyles.Active.ModalBg, content))
}
