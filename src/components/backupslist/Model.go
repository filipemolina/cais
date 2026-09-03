package backupslist

import (
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/utils"
)

// Model is the Backups page's left panel: a merged, newest-first list of
// stored versions of the compose file and the .env, each tagged with the
// source it would restore to.
//
// It owns the cursor and nothing else. The bytes behind the selected row
// belong to backuppreviewpanel, which learns about the selection through
// cmds.SetSelectedBackupMsg rather than by being handed a pointer - the two
// panels are siblings under AppModel, the same as Home's list and details.
type Model struct {
	entries     []utils.BackupEntry
	selectedIdx int
	loading     bool
	empty       bool
	loadErr     error
	panelWidth  int
	panelHeight int
}

func (m Model) Init() tea.Cmd { return nil }

// New builds the version list. It starts empty and loading; the page switch
// issues GetBackups, whose result fills it via BackupListMsg.
func New() tea.Model {
	return Model{loading: true}
}

// selected returns the entry under the cursor, or false when the list is
// empty or the cursor is out of range.
func (m Model) selected() (utils.BackupEntry, bool) {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.entries) {
		return utils.BackupEntry{}, false
	}
	return m.entries[m.selectedIdx], true
}

// publishSelection tells the preview panel which copy the cursor is on. It
// is the one place the list speaks to its sibling, so every path that moves
// the cursor goes through it rather than emitting the message by hand.
func (m Model) publishSelection() tea.Cmd {
	entry, ok := m.selected()
	if !ok {
		return cmds.ClearSelectedBackup()
	}
	return cmds.SetSelectedBackup(entry)
}
