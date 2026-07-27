package cmds

import tea "charm.land/bubbletea/v2"

type OpenHelpModalMsg struct{}

// OpenHelpModal asks AppModel to open the help overlay. Going through a
// message (rather than AppModel opening it straight from the key) is the same
// path every other modal takes, and lets anything else - a future Help button
// - open it the same way.
func OpenHelpModal() tea.Cmd {
	return func() tea.Msg { return OpenHelpModalMsg{} }
}
