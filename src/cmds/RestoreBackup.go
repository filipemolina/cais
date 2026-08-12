package cmds

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/utils"
)

// RestoreBackupMsg reports the outcome of a restore requested through the
// Backups page. Err is set when the restore failed.
type RestoreBackupMsg struct {
	Err error
}

// RestoreBackup writes the named .bak back over its live source file and
// returns a message. The source file is resolved from the Source tag:
// "compose" maps to the compose file, ".env" to the .env. The write routes
// through ReplaceFileAtomically, so the file being replaced is snapshotted
// first and the restore is itself undoable.
func RestoreBackup(source, backupName, composeFile, envPath string) tea.Cmd {
	return func() tea.Msg {
		var liveFile string
		switch source {
		case ".env":
			liveFile = envPath
		default:
			liveFile = composeFile
		}

		if liveFile == "" {
			return RestoreBackupMsg{Err: fmt.Errorf("cannot restore %q: no live file for source %q", backupName, source)}
		}

		if err := utils.RestoreBackup(liveFile, backupName); err != nil {
			return RestoreBackupMsg{Err: err}
		}

		return RestoreBackupMsg{}
	}
}
