package logsmodal

import (
	"context"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/utils"
)

type Model struct {
	viewport viewport.Model
	logCh    <-chan string
	cancel   context.CancelFunc
	lines    []string
	title    string
	follow   bool
	ended    bool
	err      error
}

func (m Model) Init() tea.Cmd {
	return nil
}

// New opens a near-full-screen overlay streaming logs for target (a
// service when isGroup is false, a group otherwise), from composeFile. It
// starts the stream immediately and returns the model plus the initial
// WaitForLog cmd; on a start failure it returns a model that just displays
// the error.
func New(target string, isGroup bool, composeFile string, termWidth, termHeight int) (tea.Model, tea.Cmd) {
	vp := viewport.New()

	m := Model{
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
