package cmds

import (
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/apptypes"
)

// SetBackupsFocusMsg names which half of the Backups page the arrows are
// driving. AppModel owns the value and tab moves it; both panels and the
// footer learn about it here, the same way SetSelectedBackupMsg carries the
// cursor from the list to the preview.
//
// Handling tab in one place rather than in both panels keeps a single owner
// for the key: two panels each deciding they now hold focus is how a page
// ends up with both halves lit or neither.
type SetBackupsFocusMsg apptypes.BackupsFocus

func SetBackupsFocus(focus apptypes.BackupsFocus) tea.Cmd {
	return func() tea.Msg {
		return SetBackupsFocusMsg(focus)
	}
}
