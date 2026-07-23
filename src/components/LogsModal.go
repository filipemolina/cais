package components

import (
	"context"
	"fmt"
	"strings"

	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/cmds"
	"stack-stitcher/src/utils"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// maxLogLines caps the in-memory scrollback so a long-running, chatty service
// can't grow the buffer without bound.
const maxLogLines = 5000

var logsModalWrapper = lipgloss.NewStyle().
	Padding(0, 1).
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(appstyles.PrimaryColor).
	Background(appstyles.PaneColor)

type LogsModalModel struct {
	viewport viewport.Model
	logCh    <-chan string
	cancel   context.CancelFunc
	lines    []string
	title    string
	follow   bool
	ended    bool
	err      error
}

func (m LogsModalModel) Init() tea.Cmd {
	return nil
}

func (m LogsModalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		switch msg.String() {
		case "esc":
			if m.cancel != nil {
				m.cancel()
			}
			return m, cmds.CloseModal(nil)

		case "f":
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

// resize recomputes the viewport dimensions from the current terminal size,
// leaving room for the wrapper chrome plus the title and footer lines.
func (m *LogsModalModel) resize(termWidth, termHeight int) {
	width := int(float32(termWidth) * 0.9)
	height := int(float32(termHeight) * 0.9)

	h, v := logsModalWrapper.GetFrameSize()
	// Reserve two rows for the title and the footer hint.
	m.viewport.SetWidth(max(1, width-h))
	m.viewport.SetHeight(max(1, height-v-2))
}

func (m LogsModalModel) View() tea.View {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(appstyles.PrimaryFontColor).
		Render("logs: " + m.title)

	followState := "off"
	if m.follow {
		followState = "on"
	}
	hint := fmt.Sprintf("↑↓ scroll · f follow (%s) · esc quit", followState)
	if m.ended {
		hint = "stream ended · " + hint
	}
	footer := lipgloss.NewStyle().
		Foreground(appstyles.SecondaryFontColor).
		Render(hint)

	body := m.viewport.View()
	if m.err != nil {
		body = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Render("Error: " + m.err.Error())
	}

	content := lipgloss.JoinVertical(lipgloss.Left, title, body, footer)
	return tea.NewView(logsModalWrapper.Render(content))
}

// LogsModal opens a near-full-screen overlay streaming logs for target (a
// service when isProfile is false, a profile otherwise). It starts the stream
// immediately and returns the model plus the initial WaitForLog cmd; on a
// start failure it returns a model that just displays the error.
func LogsModal(target string, isProfile bool, termWidth, termHeight int) (tea.Model, tea.Cmd) {
	vp := viewport.New()

	m := LogsModalModel{
		viewport: vp,
		title:    target,
		follow:   true,
	}
	m.resize(termWidth, termHeight)

	ch, cancel, err := utils.StreamDockerLogs(target, isProfile)
	if err != nil {
		m.err = err
		return m, nil
	}

	m.logCh = ch
	m.cancel = cancel

	return m, cmds.WaitForLog(ch)
}
