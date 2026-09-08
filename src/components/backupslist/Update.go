package backupslist

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/keys"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var finalCmds []tea.Cmd

	// See serviceslist.Model.Update: the footer's keys depend on this, so a
	// change has to be broadcast.
	filterStateBefore := m.list.FilterState()

	// Note which copy the cursor is on before the list processes the message,
	// so a row changing underneath it can be told apart from everything else.
	previous, hadSelection := m.selected()

	switch msg := msg.(type) {
	// Sizing comes from AppModel like every other panel; deriving it from
	// WindowSizeMsg here would leave the panel at width 0 whenever Backups
	// wasn't the active page at resize time.
	case cmds.SetBodyLayoutMsg:
		m.panelWidth = msg.LeftWidth
		m.panelHeight = msg.Height
		m.resizeList()

	// Focus is AppModel's to own - tab is handled once, up there - so this is
	// the panel learning the answer rather than deciding it.
	case cmds.SetBackupsFocusMsg:
		m.setFocus(apptypes.BackupsFocus(msg) == apptypes.BackupsList)

	case cmds.BackupListMsg:
		if msg.Err != nil {
			m.loadErr = msg.Err
			m.loading = false
			m.rowsRebuilding = false
			finalCmds = append(finalCmds, m.setItems(nil))

			return m, tea.Batch(append(finalCmds, cmds.ClearSelectedBackup())...)
		}

		m.loading = false
		m.loadErr = nil

		// Which copy is the live file's twin, decided once per read: a row
		// is current when its own content hash matches its source's live
		// hash. The lookup is keyed by Source, so a compose copy cannot
		// match a .env's hash by coincidence.
		currentBySource := make(map[string]string, len(msg.Live))
		for _, live := range msg.Live {
			currentBySource[live.Source] = live.SHA8
		}

		items := make([]list.Item, 0, len(msg.Entries))
		for _, entry := range msg.Entries {
			items = append(items, backupItem{
				entry:     entry,
				isCurrent: entry.SHA8 != "" && currentBySource[entry.Source] == entry.SHA8,
			})
		}

		// A reload re-lists the store, so the cursor goes back to the newest
		// copy. After a restore that is the copy the restore just made of the
		// live file, which is the one worth landing on.
		finalCmds = append(finalCmds, m.setItems(items))
		m.list.Select(0)

		// SetItems blanks the filtered rows and only re-applies the filter a
		// message later, so under a standing filter the cursor is over nothing
		// for one cycle. Publishing that would clear the preview and then fill
		// it again a frame later; the flag suspends the auto-publish until the
		// rows come back. Nothing sets it on the unfiltered path, which is the
		// normal one.
		m.rowsRebuilding = m.list.FilterState() != list.Unfiltered

	case tea.KeyPressMsg:
		// The inner list still gets the key below - that is where the filter
		// input lives - but the restore verb does not fire while it is being
		// typed into. Same rule as the services list's d.
		if m.OwnsKeyboard() {
			break
		}

		// Restore is matched ahead of the focus gate because it does not
		// belong to either panel: focus decides what the arrows drive, and the
		// arrows never move the selection out from under it. r acts on the row
		// the cursor is on, and that row is the same whichever half is lit.
		if key.Matches(msg, keys.Backup.Restore) {
			if entry, ok := m.selected(); ok {
				finalCmds = append(finalCmds, cmds.RequestRestoreBackup(entry.Source, entry.Name))
			}
		}
	}

	// The list is handed every message except a keystroke meant for the
	// preview. Focus gates only the keys: a resize or a rebuild has to reach
	// the list whichever panel is lit, and withholding it would leave an
	// unfocused list sized for the previous terminal.
	if !m.swallowsKey(msg) {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		finalCmds = append(finalCmds, cmd)
	}

	// The rows are back, so the cursor means something again.
	if m.rowsRebuilding && len(m.list.VisibleItems()) > 0 {
		m.rowsRebuilding = false
	}

	// Auto-select: if a different copy is under the cursor now, publish it.
	//
	// The test is the .bak's name, not list.Index(). Filtering moves rows out
	// from under a cursor that never moves - applying a filter sends the
	// cursor to the top, so a cursor already at the top keeps the same index
	// while row 0 becomes an entirely different copy. See
	// serviceslist.Model.Update, which had the same bug against indexes.
	if !m.rowsRebuilding {
		current, hasSelection := m.selected()
		if current.Name != previous.Name || hasSelection != hadSelection {
			finalCmds = append(finalCmds, m.publishSelection())
		}
	}

	filterState := m.list.FilterState()
	if filterState != filterStateBefore {
		finalCmds = append(finalCmds, cmds.SetListFilterState(filterState))
	}

	return m, tea.Batch(finalCmds...)
}

// swallowsKey reports whether this message is a keystroke the list must not
// see. Everything the preview drives while it holds focus - the arrows, the
// paging keys, the jumps - is also a key this list would answer, and both
// panels receive every message, so one of them has to decline.
//
// A filter being typed is the exception: the list owns the whole keyboard
// then, and it cannot lose focus mid-filter, so the keystrokes are its own.
func (m Model) swallowsKey(msg tea.Msg) bool {
	if _, ok := msg.(tea.KeyPressMsg); !ok {
		return false
	}

	return !m.focused && !m.OwnsKeyboard()
}
