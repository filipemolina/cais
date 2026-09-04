package backupslist

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/keys"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// Sizing comes from AppModel like every other panel; deriving it from
	// WindowSizeMsg here would leave the panel at width 0 whenever Backups
	// wasn't the active page at resize time.
	case cmds.SetBodyLayoutMsg:
		m.setSize(msg.LeftWidth, msg.Height)
		return m, nil

	case cmds.BackupListMsg:
		if msg.Err != nil {
			m.loadErr = msg.Err
			m.loading = false
			m.empty = false
			m.entries = nil
			m.selectedIdx = 0
			m.rowOffset = 0
			return m, cmds.ClearSelectedBackup()
		}
		m.entries = msg.Entries
		m.loading = false
		m.loadErr = nil
		m.empty = len(m.entries) == 0
		// A reload re-lists the store, so the cursor goes back to the newest
		// copy and the rows scroll back to the top with it.
		m.selectedIdx = 0
		m.rowOffset = 0
		return m, m.publishSelection()

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if m.loading || m.loadErr != nil || len(m.entries) == 0 {
		return m, nil
	}

	switch {
	// Matched against real keys rather than keys.Backup.Navigate, which is a
	// help-only binding: it carries a label for the footer and no keys at
	// all, so key.Matches against it can never be true. This panel hand-rolls
	// its list instead of embedding a bubbles one, so nothing else is moving
	// the cursor.
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if m.selectedIdx > 0 {
			return m, m.moveCursorTo(m.selectedIdx - 1)
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		if m.selectedIdx < len(m.entries)-1 {
			return m, m.moveCursorTo(m.selectedIdx + 1)
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("home", "g"))):
		return m, m.moveCursorTo(0)

	case key.Matches(msg, key.NewBinding(key.WithKeys("end", "G"))):
		return m, m.moveCursorTo(len(m.entries) - 1)

	case key.Matches(msg, keys.Backup.Restore):
		entry, ok := m.selected()
		if !ok {
			return m, nil
		}
		return m, cmds.RequestRestoreBackup(entry.Source, entry.Name)
	}

	return m, nil
}
