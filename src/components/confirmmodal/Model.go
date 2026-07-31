package confirmmodal

import (
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

type Model struct {
	message string
	confirm tea.Cmd
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch {
	case key.Matches(keyMsg, keys.Overlay.Yes):
		return m, cmds.CloseModal(m.confirm)
	case key.Matches(keyMsg, keys.Overlay.No, keys.Overlay.Cancel):
		return m, cmds.CloseModal(nil)
	}

	return m, nil
}

func (m Model) View() tea.View {
	// The hint line is where y/n is advertised, so the messages callers pass
	// in are plain questions - they used to each spell out "(y/n)" and none of
	// them mentioned that esc also backs out.
	content := lipgloss.JoinVertical(lipgloss.Left,
		chrome.ModalTitle("Confirm"),
		m.message,
		"",
		chrome.ModalHints(
			chrome.HintFor(keys.Overlay.Yes),
			chrome.HintFor(keys.Overlay.No),
			chrome.HintFor(keys.Overlay.Cancel),
		),
	)

	return tea.NewView(chrome.ModalSurface(appstyles.Active.ModalBg, content))
}

// New shows message and, if the user presses 'y', runs confirm once the
// modal closes. 'n' or Esc dismisses without running it.
func New(message string, confirm tea.Cmd) tea.Model {
	return Model{
		message: message,
		confirm: confirm,
	}
}
