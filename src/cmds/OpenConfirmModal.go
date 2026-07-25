package cmds

import tea "charm.land/bubbletea/v2"

type OpenConfirmModalMsg struct {
	Message string
	Follow  tea.Cmd
}

// OpenConfirmModal asks AppModel to show a yes/no confirmation dialog.
// Follow runs only if the user confirms.
func OpenConfirmModal(message string, follow tea.Cmd) tea.Cmd {
	return func() tea.Msg {
		return OpenConfirmModalMsg{Message: message, Follow: follow}
	}
}
