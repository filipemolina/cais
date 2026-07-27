package components

import (
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/cmds"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

type ConfirmModalModel struct {
	message string
	confirm tea.Cmd
}

func (m ConfirmModalModel) Init() tea.Cmd {
	return nil
}

func (m ConfirmModalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m ConfirmModalModel) View() tea.View {
	return tea.NewView(modalSurface(appstyles.Active.ModalBg, m.message))
}

// ConfirmModal shows message and, if the user presses 'y', runs confirm
// once the modal closes. 'n' or Esc dismisses without running it.
func ConfirmModal(message string, confirm tea.Cmd) tea.Model {
	return ConfirmModalModel{
		message: message,
		confirm: confirm,
	}
}
