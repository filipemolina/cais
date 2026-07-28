package cmds

import tea "charm.land/bubbletea/v2"

// OpenErrorModalMsg opens an error modal with the given message.
// Used for foreground errors (docker actions, config loads) where a modal
// is less disruptive than the banner. Background poll errors keep the banner.
type OpenErrorModalMsg struct {
	Message string
}

// OpenErrorModal returns a command that opens an error modal.
func OpenErrorModal(message string) tea.Cmd {
	return func() tea.Msg {
		return OpenErrorModalMsg{Message: message}
	}
}
