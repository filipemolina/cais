package groupnamemodal

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

func (m Model) View() tea.View {
	title := "New group"
	submitDesc := "next"
	if m.isRename {
		title = "Rename group"
		submitDesc = "rename"
	}

	lines := []string{chrome.ModalTitle(title), "Group name:", m.input.View()}
	if m.errMsg != "" {
		errStyle := lipgloss.NewStyle().Foreground(appstyles.Active.Danger)
		lines = append(lines, errStyle.Render(m.errMsg))
	}

	// Enter is "next" on the create flow (step 1 of two, handing off to
	// the service checklist rather than writing anything) and "rename" on
	// the rename flow (the only step, which writes).
	lines = append(lines, "", chrome.ModalHints(
		chrome.HintAs(keys.Overlay.Submit, submitDesc),
		chrome.HintFor(keys.Overlay.Cancel),
	))

	return tea.NewView(chrome.ModalSurface(
		appstyles.Active.ModalBg,
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	))
}
