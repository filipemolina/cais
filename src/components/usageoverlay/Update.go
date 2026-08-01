package usageoverlay

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height

	case tea.KeyPressMsg:
		// Close on u (the key that opened it), esc, or q.
		switch {
		case key.Matches(msg, keys.Global.Usage),
			key.Matches(msg, keys.Overlay.Cancel),
			key.Matches(msg, keys.Global.Quit):
			return m, cmds.CloseModal(nil)
		// Refresh on r.
		case key.Matches(msg, keys.Details.Restart):
			if !m.loading {
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, cmds.GetDockerUsage(m.containers))
			}
		}

	case cmds.DockerUsageMsg:
		m.disk = msg.Disk
		m.memTotal = msg.MemTotal
		m.containers = msg.Containers
		m.err = msg.Err
		m.loading = false
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}