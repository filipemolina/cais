package backuppreviewpanel

import (
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/highlight"
	"github.com/filipemolina/cais/src/utils"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case cmds.SetBodyLayoutMsg:
		m.setSize(msg.RightWidth, msg.Height)
		return m, nil

	case cmds.SetSelectedBackupMsg:
		entry := utils.BackupEntry(msg)
		m.entry = entry
		m.loadErr = nil
		if entry.Name == "" {
			m.content = ""
			m.vp.SetContent("")
			return m, nil
		}
		// The bytes are stale the moment the selection moves, so clear them
		// rather than leaving the previous copy on screen under the new
		// row's header while the read is in flight.
		m.content = ""
		m.vp.SetContent("")
		return m, readBackupCmd(entry)

	case backupContentsMsg:
		// A read that finished after the cursor moved on belongs to a row
		// that is no longer selected; dropping it keeps the panel showing
		// what its header names.
		if msg.Path != m.entry.Path {
			return m, nil
		}
		if msg.Err != nil {
			m.loadErr = msg.Err
			return m, nil
		}
		m.setContent(msg.Source, msg.Contents)
		return m, nil
	}

	return m, nil
}

// backupContentsMsg carries the selected .bak's bytes for the preview. Path
// identifies which read came back, so a stale one can be discarded.
type backupContentsMsg struct {
	Path     string
	Source   string
	Contents []byte
	Err      error
}

// readBackupCmd reads the selected .bak's bytes off the disk.
func readBackupCmd(entry utils.BackupEntry) tea.Cmd {
	return func() tea.Msg {
		contents, err := os.ReadFile(entry.Path)
		if err != nil {
			return backupContentsMsg{Path: entry.Path, Source: entry.Source, Err: err}
		}
		return backupContentsMsg{Path: entry.Path, Source: entry.Source, Contents: contents}
	}
}

// setContent renders the copy for the viewport: compose copies get YAML
// syntax highlighting, .env copies are shown raw so secrets are not masked
// and the raw line is exactly what a restore would write.
func (m *Model) setContent(source string, contents []byte) {
	raw := string(contents)
	m.content = raw
	if source == "compose" {
		m.vp.SetContent(highlight.YAML(raw))
	} else {
		m.vp.SetContent(raw)
	}
	m.vp.GotoTop()
}
