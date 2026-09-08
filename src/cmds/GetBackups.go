package cmds

import (
	"os"
	"path/filepath"
	"sort"

	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/utils"
)

// BackupListMsg carries the merged list of stored versions for the live
// sources the Backups page covers: the compose file and the .env (when one
// is known). Source names which file each row restores, so a single list can
// mix both without losing track of where the copy came from.
//
// The same read also carries each live file's current state, once: the list
// marks the copies that match what is on disk right now, and the preview
// diffs a copy against it. Reading the live files here - once per list load,
// from the paths AppModel resolved - is what keeps the panels from needing
// to know which files are loaded, or from re-reading disk per row.
type BackupListMsg struct {
	// Source is the file the list was read for, kept for callers that want
	// to distinguish the page-level read from a single-source re-list.
	Source string
	// Entries is newest-first, merged across compose and .env.
	Entries []utils.BackupEntry
	// Live is the current state of each source read for, keyed by the same
	// Source label the entries carry. A live file that could not be read
	// (gone from disk, or a .env that does not exist yet) contributes
	// nothing: there is nothing to diff against, and the panels already
	// handle that answer.
	Live []utils.LiveSource
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
		var liveReads []utils.LiveSource

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
			if live, ok := liveSourceFor(src); ok {
				liveReads = append(liveReads, live)
			}
		}

		// Merge stays newest-first: names sort lexically by UTC prefix, and
		// the newest is last, so a descending sort across both sources keeps
		// the most recent copy on top regardless of which file it belongs to.
		sort.SliceStable(merged, func(i, j int) bool {
			return merged[i].Name > merged[j].Name
		})

		return BackupListMsg{Entries: merged, Live: liveReads}
	}
}

// liveSourceFor reads one live file's current state for the list message. A
// file that has gone away - or a .env that was never written - is not an
// error: the store still lists, and "no live bytes" is a state the panels
// already answer for, the same as a slug folder that does not exist. Only a
// file that is there and readable has a state worth carrying.
func liveSourceFor(src string) (utils.LiveSource, bool) {
	contents, err := os.ReadFile(src)
	if err != nil {
		return utils.LiveSource{}, false
	}

	return utils.LiveSource{
		Source:   utils.BackupSourceLabel(src),
		File:     filepath.Base(src),
		SHA8:     utils.ContentSHA8(contents),
		Contents: string(contents),
	}, true
}
