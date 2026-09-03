package cmds

import (
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/utils"
)

// SetSelectedBackupMsg names the stored copy the Backups list's cursor is
// on. The list owns the cursor and the preview panel owns the viewport, so
// this is how the two halves of the page agree on what is being shown -
// the same shape as SetSelectedGroupMsg between Home's two panels.
//
// The zero value means nothing is selected: an empty Name is what the
// preview shows its empty state for, which is the state an empty store and
// a failed read both land in.
type SetSelectedBackupMsg utils.BackupEntry

func SetSelectedBackup(entry utils.BackupEntry) tea.Cmd {
	return func() tea.Msg {
		return SetSelectedBackupMsg(entry)
	}
}

// ClearSelectedBackup is SetSelectedBackup's empty case, spelled out so a
// caller reads as clearing rather than as sending a blank entry.
func ClearSelectedBackup() tea.Cmd {
	return SetSelectedBackup(utils.BackupEntry{})
}
