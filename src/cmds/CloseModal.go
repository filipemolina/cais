package cmds

import tea "charm.land/bubbletea/v2"

// CloseModalMsg tells AppModel to clear the active modal. Follow, if set,
// is appended to the batch of commands run once the modal is gone - this is
// how a modal hands off the action it collected input for (e.g. actually
// creating a profile) without needing to know about AppModel itself.
type CloseModalMsg struct {
	Follow tea.Cmd
}

func CloseModal(follow tea.Cmd) tea.Cmd {
	return func() tea.Msg { return CloseModalMsg{Follow: follow} }
}
