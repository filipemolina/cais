package cmds

import tea "charm.land/bubbletea/v2"

// OpenAboutModalMsg asks AppModel to open the About modal. Going through a
// message (rather than AppModel opening it straight from the key) is the same
// path every other modal takes.
type OpenAboutModalMsg struct{}

// OpenAboutModal opens the About modal: the brand mark, version, license and
// repo link. It is a read-only overlay, like the help overlay.
func OpenAboutModal() tea.Cmd {
	return func() tea.Msg { return OpenAboutModalMsg{} }
}
