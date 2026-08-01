package envkeymodal

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

func (m Model) View() tea.View {
	title := "Add Variable"
	if m.isEdit {
		title = "Edit Variable"
	}

	sections := []string{
		chrome.ModalTitle(title),
		"Variable Name:",
		m.keyInput.View(),
		"",
		"Variable Value:",
		m.valueInput.View(),
	}

	if m.err != nil {
		sections = append(sections, lipgloss.NewStyle().
			Foreground(appstyles.Active.Danger).
			Render("Error: "+m.err.Error()))
	}

	sections = append(sections,
		"",
		chrome.ModalHints(
			chrome.HintFor(keys.Overlay.Submit),
			chrome.HintFor(keys.Overlay.NextField),
			chrome.HintFor(keys.Overlay.Cancel),
		),
	)

	content := strings.Join(sections, "\n")
	return tea.NewView(chrome.ModalSurface(appstyles.Active.ModalBg, content))
}
