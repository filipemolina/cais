package cmds

import (
	tea "charm.land/bubbletea/v2"

	"github.com/filipemolina/cais/src/utils"
)

// SetImageUpdatesMsg hands the panels the current image update map, the way
// SetGroupsListMsg hands the lists their status dots. It rides config loads
// and page switches, so a panel that was not active when the check answered
// still gets its glyphs the next time it is shown.
type SetImageUpdatesMsg struct {
	Updates map[string]utils.ImageUpdate
}

// SetImageUpdates returns a command that broadcasts the update map to the
// active page's panels.
func SetImageUpdates(updates map[string]utils.ImageUpdate) tea.Cmd {
	return func() tea.Msg {
		return SetImageUpdatesMsg{Updates: updates}
	}
}
