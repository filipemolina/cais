package components

import (
	"context"
	"fmt"
	"strings"

	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/utils"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

// maxLogLines caps the in-memory scrollback so a long-running, chatty service
// can't grow the buffer without bound.
const maxLogLines = 5000

// logsModalWrapper builds the near-full-screen overlay's chrome fresh each
// call, so it re-reads appstyles.Active instead of freezing whichever theme
// was active when the package loaded.
//
// BorderBackground matters as much as Background here: without it lipgloss
// leaves the border cells on the terminal's default color, outlining a
// near-full-screen overlay in the wrong shade.
func logsModalWrapper() lipgloss.Style {
	return lipgloss.NewStyle().
		Padding(0, 1).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(appstyles.Active.Accent).
		BorderBackground(appstyles.Active.PanelBg).
		Background(appstyles.Active.PanelBg)
}

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

// resize recomputes the viewport dimensions from the current terminal size,
// leaving room for the wrapper chrome plus the title and footer lines.
func (m *LogsModalModel) resize(termWidth, termHeight int) {
	width := int(float32(termWidth) * 0.9)
	height := int(float32(termHeight) * 0.9)

	h, v := logsModalWrapper().GetFrameSize()
	// Reserve two rows for the title and the footer hint.
	m.viewport.SetWidth(max(1, width-h))
	m.viewport.SetHeight(max(1, height-v-2))
}

func (m LogsModalModel) View() tea.View {
	title := chrome.ModalTitle("logs: " + m.title)

	followState := "off"
	if m.follow {
		followState = "on"
	}

	// Built from the bindings rather than written out, so rebinding follow or
	// cancel cannot leave this line advertising the old key.
	footer := chrome.RenderKeyHints([]chrome.KeyHint{
		chrome.HintAs(keys.List.Navigate, "scroll"),
		chrome.HintAs(keys.Overlay.Follow, fmt.Sprintf("follow (%s)", followState)),
		chrome.HintAs(keys.Overlay.Cancel, "quit"),
	}, appstyles.Active.TextMuted)

	if m.ended {
		footer = lipgloss.NewStyle().
			Foreground(appstyles.Active.TextMuted).
			Render("stream ended · ") + footer
	}

	body := m.viewport.View()
	if m.err != nil {
		body = lipgloss.NewStyle().
			Foreground(appstyles.Active.TextPrimary).
			Render("Error: " + m.err.Error())
	}

	// The title and footer are far shorter than the viewport, so JoinVertical
	// pads them out with unstyled spaces; seal them against the modal's
	// background before the wrapper draws its border.
	content := appstyles.FillBackground(
		appstyles.Active.PanelBg,
		lipgloss.JoinVertical(lipgloss.Left, title, body, footer),
	)

	return tea.NewView(logsModalWrapper().Render(content))
}

// LogsModal opens a near-full-screen overlay streaming logs for target (a
// service when isGroup is false, a group otherwise), from composeFile. It
// starts the stream immediately and returns the model plus the initial
// WaitForLog cmd; on a start failure it returns a model that just displays
// the error.
func LogsModal(target string, isGroup bool, composeFile string, termWidth, termHeight int) (tea.Model, tea.Cmd) {
	vp := viewport.New()

	m := LogsModalModel{
		viewport: vp,
		title:    target,
		follow:   true,
	}
	m.resize(termWidth, termHeight)

	ch, cancel, err := utils.StreamDockerLogs(target, isGroup, composeFile)
	if err != nil {
		m.err = err
		return m, nil
	}

	m.logCh = ch
	m.cancel = cancel

	return m, cmds.WaitForLog(ch)
}
