package createcomposefilemodal

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

func (m Model) View() tea.View {
	if m.step == stepServiceFields {
		return tea.NewView(chrome.ModalSurface(appstyles.Active.ModalBg, m.serviceStep.View()))
	}

	errStyle := lipgloss.NewStyle().Foreground(appstyles.Active.Danger)
	var lines []string
	var hints string

	// The title names the file being built once there is one to name, so the
	// two later steps say which file they are filling in rather than just
	// "New compose file". Each step advertises only the keys it answers -
	// tab means nothing on the y/n step, and enter means nothing either.
	switch m.step {
	case stepFilename:
		lines = []string{
			chrome.ModalTitle("New compose file"),
			"Filename (in the current directory):",
			m.filename.View(),
		}
		hints = chrome.ModalHints(
			chrome.HintAs(keys.Overlay.Submit, "next"),
			chrome.HintFor(keys.Overlay.Cancel),
		)
	case stepAddServicePrompt:
		lines = []string{
			chrome.ModalTitle(fmt.Sprintf("Creating %s", filepath.Base(strings.TrimSpace(m.filename.Value())))),
			"Add a first service?",
		}
		hints = chrome.ModalHints(
			chrome.HintAs(keys.Overlay.Yes, "add a service"),
			chrome.HintAs(keys.Overlay.No, "skip"),
			chrome.HintFor(keys.Overlay.Cancel),
		)
	}

	if m.errMsg != "" {
		lines = append(lines, errStyle.Render(m.errMsg))
	}

	lines = append(lines, "", hints)

	return tea.NewView(chrome.ModalSurface(
		appstyles.Active.ModalBg,
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	))
}
