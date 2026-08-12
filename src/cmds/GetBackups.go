package cmds

import (
	"sort"

	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/utils"
)

// BackupListMsg carries the merged list of stored versions for the live
// sources the Backups page covers: the compose file and the .env (when one
// is known). Source names which file each row restores, so a single list can
// mix both without losing track of where the copy came from.
type BackupListMsg struct {
	// Source is the file the list was read for, kept for callers that want
	// to distinguish the page-level read from a single-source re-list.
	Source string
	// Entries is newest-first, merged across compose and .env.
	Entries []utils.BackupEntry
	// Err is set when the store could not be read.
	Err error
}

// GetBackups reads the backup store for each live source that has a path and
// merges them newest-first into one list. A source whose slug folder does not
// exist (never written, or brand-new) simply contributes nothing, so a stack
// with no .env yet still lists its compose backups.
func GetBackups(composeFile, envPath string) tea.Cmd {
	return func() tea.Msg {
		var merged []utils.BackupEntry

		sources := []string{composeFile}
		if envPath != "" {
			sources = append(sources, envPath)
		}

		for _, src := range sources {
			if src == "" {
				continue
			}
			entries, err := utils.ListBackups(src)
			if err != nil {
				return BackupListMsg{Err: err}
			}
			merged = append(merged, entries...)
		}

		// Merge stays newest-first: names sort lexically by UTC prefix, and
		// the newest is last, so a descending sort across both sources keeps
		// the most recent copy on top regardless of which file it belongs to.
		sort.SliceStable(merged, func(i, j int) bool {
			return merged[i].Name > merged[j].Name
		})

		return BackupListMsg{Entries: merged}
	}
}
