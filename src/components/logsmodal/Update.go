package logsmodal

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

// maxLogLines caps the in-memory scrollback so a long-running, chatty service
// can't grow the buffer without bound.
const maxLogLines = 5000

// resize recomputes the viewport dimensions from the current terminal size,
// leaving room for the wrapper chrome plus the title and footer lines.
func (m *Model) resize(termWidth, termHeight int) {
	width := int(float32(termWidth) * 0.9)
	height := int(float32(termHeight) * 0.9)

	h, v := logsModalWrapper().GetFrameSize()
	// Reserve two rows for the title and the footer hint.
	m.viewport.SetWidth(max(1, width-h))
	m.viewport.SetHeight(max(1, height-v-2))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case cmds.LogLineMsg:
		m.lines = append(m.lines, string(msg))
		if len(m.lines) > maxLogLines {
			m.lines = m.lines[len(m.lines)-maxLogLines:]
		}
		m.viewport.SetContent(strings.Join(m.lines, "\n"))
		if m.follow {
			m.viewport.GotoBottom()
		}
		// Pull the next line to keep the stream flowing.
		return m, cmds.WaitForLog(m.logCh)

	case cmds.LogStreamEndedMsg:
		m.ended = true
		return m, nil

	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keys.Overlay.Cancel):
			if m.cancel != nil {
				m.cancel()
			}
			return m, cmds.CloseModal(nil)

		case key.Matches(msg, keys.Overlay.Follow):
			m.follow = !m.follow
			if m.follow {
				m.viewport.GotoBottom()
			}
			return m, nil
		}

		// Any other key (scroll navigation) goes to the viewport; keep follow
		// in sync with whether we're pinned to the bottom.
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		m.follow = m.viewport.AtBottom()
		return m, cmd
	}

	return m, nil
}
