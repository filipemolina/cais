package components

import (
	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/cmds"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

	switch keyMsg.String() {
	case "y":
		return m, cmds.CloseModal(m.confirm)
	case "n", "esc":
		return m, cmds.CloseModal(nil)
	}

	return m, nil
}

func (m ConfirmModalModel) View() tea.View {
	style := lipgloss.NewStyle().
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(appstyles.PrimaryColor).
		Background(appstyles.PanelBackgroundColor)

	return tea.NewView(style.Render(m.message))
}

// ConfirmModal shows message and, if the user presses 'y', runs confirm
// once the modal closes. 'n' or Esc dismisses without running it.
func ConfirmModal(message string, confirm tea.Cmd) tea.Model {
	return ConfirmModalModel{
		message: message,
		confirm: confirm,
	}
}
