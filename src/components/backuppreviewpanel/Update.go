package backuppreviewpanel

import (
	"os"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/apptypes"
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

	// Focus is AppModel's to own - tab is handled once, up there - so this is
	// the panel learning the answer rather than deciding it.
	case cmds.SetBackupsFocusMsg:
		m.focused = apptypes.BackupsFocus(msg) == apptypes.BackupsPreview
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

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

// handleKey scrolls the file while this panel holds focus. Unfocused it
// answers nothing: the same keystroke is reaching the list beside it, and one
// arrow press must move one thing.
//
// r is not matched here even though it is live on either half of the page. The
// list owns the cursor and therefore owns the restore, and it sees this
// keystroke too - both panels are updated with every message.
func (m Model) handleKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	// g/G and home/end are matched here rather than left to the viewport: its
	// keymap has no top/bottom binding at all, and the list answers the same
	// two keys with first/last entry, so without these the preview would be
	// the one panel where G does nothing.
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("home", "g"))):
		m.vp.GotoTop()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("end", "G"))):
		m.vp.GotoBottom()
		return m, nil
	}

	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)

	return m, cmd
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
//
// "Raw" is about the bytes, not the colour. An .env copy still gets a
// foreground - see plainText.
func (m *Model) setContent(source string, contents []byte) {
	raw := string(contents)
	m.content = raw
	if source == "compose" {
		m.vp.SetContent(highlight.YAML(raw))
	} else {
		m.vp.SetContent(plainText(raw))
	}
	m.vp.GotoTop()
}

// plainText gives unhighlighted content an explicit foreground.
//
// Without one the text carries no SGR at all and lands on the terminal's own
// default foreground, which has nothing to do with the active theme: on a
// light theme the .env preview was pale grey on a pale panel and effectively
// unreadable. Every other body of text in the app names its colour, and
// highlight.YAML does it per line for compose copies; this is the same for the
// copies that get no syntax highlighting.
//
// Styled per line rather than over the whole block because lipgloss closes
// each styled run with a reset, so a single Render of a multi-line string
// leaves everything after the first newline unstyled again.
func plainText(content string) string {
	if content == "" {
		return content
	}

	style := lipgloss.NewStyle().Foreground(appstyles.Active.TextPrimary)

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		lines[i] = style.Render(line)
	}

	return strings.Join(lines, "\n")
}
