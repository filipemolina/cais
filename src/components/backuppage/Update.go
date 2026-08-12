package backuppage

import (
	"os"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/constants"
	"github.com/filipemolina/cais/src/highlight"
	"github.com/filipemolina/cais/src/keys"
	"github.com/filipemolina/cais/src/utils"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// Sizing comes from AppModel like every other panel; deriving it from
	// WindowSizeMsg here would leave the panel at width 0 whenever Backups
	// wasn't the active page at resize time. As the sole component on its
	// page it takes the whole body row: both panel widths plus the gutter.
	case cmds.SetBodyLayoutMsg:
		m.setSize(msg.LeftWidth+constants.BODY_GUTTER_WIDTH+msg.RightWidth, msg.Height)
		return m, nil

	case cmds.BackupListMsg:
		if msg.Err != nil {
			m.loadErr = msg.Err
			m.loading = false
			m.empty = false
			return m, nil
		}
		m.entries = msg.Entries
		m.loading = false
		m.loadErr = nil
		m.empty = len(m.entries) == 0
		m.selectedIdx = 0
		// Load the first copy into the preview when there is one.
		if len(m.entries) > 0 {
			return m, m.loadPreviewCmd(m.entries[0])
		}
		m.preview = ""
		m.previewVP.SetContent("")
		return m, nil

	case backupPreviewMsg:
		if msg.Err == nil {
			m.setPreview(msg.Source, msg.Contents)
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	// Hand everything else (scroll navigation on the preview viewport) to
	// the viewport.
	var cmd tea.Cmd
	m.previewVP, cmd = m.previewVP.Update(msg)
	return m, cmd
}

func (m Model) handleKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if m.loading || m.loadErr != nil {
		return m, nil
	}
	if len(m.entries) == 0 {
		return m, nil
	}

	switch {
	case key.Matches(msg, keys.List.Navigate):
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if m.selectedIdx > 0 {
				m.selectedIdx--
				return m, m.loadPreviewCmd(m.entries[m.selectedIdx])
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			if m.selectedIdx < len(m.entries)-1 {
				m.selectedIdx++
				return m, m.loadPreviewCmd(m.entries[m.selectedIdx])
			}
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("home", "g"))):
		m.selectedIdx = 0
		return m, m.loadPreviewCmd(m.entries[m.selectedIdx])

	case key.Matches(msg, key.NewBinding(key.WithKeys("end", "G"))):
		m.selectedIdx = len(m.entries) - 1
		return m, m.loadPreviewCmd(m.entries[m.selectedIdx])

	case key.Matches(msg, keys.Backup.Restore):
		entry := m.entries[m.selectedIdx]
		return m, cmds.RequestRestoreBackup(entry.Source, entry.Name)
	}

	return m, nil
}

// loadPreviewCmd reads the selected .bak's bytes off the disk and returns
// them as a backupPreviewMsg, which the model renders into the preview.
func (m Model) loadPreviewCmd(entry utils.BackupEntry) tea.Cmd {
	return func() tea.Msg {
		contents, err := os.ReadFile(entry.Path)
		if err != nil {
			return backupPreviewMsg{Source: entry.Source, Err: err}
		}
		return backupPreviewMsg{Source: entry.Source, Contents: contents}
	}
}

// backupPreviewMsg carries the selected .bak's bytes for the preview.
type backupPreviewMsg struct {
	Source   string
	Contents []byte
	Err      error
}

// setPreview renders the copy's content for the preview viewport: compose
// copies get YAML syntax highlighting, .env copies are shown raw so secrets
// are not masked and the raw line is exactly what a restore would write.
func (m *Model) setPreview(source string, contents []byte) {
	raw := string(contents)
	m.preview = raw
	if source == "compose" {
		m.previewVP.SetContent(highlight.YAML(raw))
	} else {
		m.previewVP.SetContent(raw)
	}
	m.previewVP.GotoTop()
}
