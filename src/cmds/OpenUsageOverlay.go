package cmds

import tea "charm.land/bubbletea/v2"

// OpenUsageOverlayMsg asks AppModel to open the Usage overlay. Going through a
// message (rather than AppModel opening it straight from the key) is the same
// path every other modal takes.
type OpenUsageOverlayMsg struct{}

// OpenUsageOverlay opens the Usage overlay: disk and memory usage bars.
func OpenUsageOverlay() tea.Cmd {
	return func() tea.Msg { return OpenUsageOverlayMsg{} }
}
