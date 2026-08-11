package groupdetailspanel

import (
	"fmt"

	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/chrome"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/keys"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var finalCmds []tea.Cmd

	switch msg := msg.(type) {
	// AppModel emits this on resize, page switch, and error-banner changes,
	// so the panel always fills the exact body region. Deriving the width
	// from WindowSizeMsg instead would leave it at 0 until Home happened to
	// be the active page during a resize.
	case cmds.SetBodyLayoutMsg:
		m.panelWidth = msg.RightWidth
		m.panelHeight = msg.Height

	case cmds.SetPendingActionMsg:
		m.pendingAction = &chrome.PendingAction{Action: msg.Action, Target: msg.Target, IsGroup: msg.IsGroup}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(m.spinner.Tick())
		finalCmds = append(finalCmds, cmd)

	case cmds.ClearPendingActionMsg:
		m.pendingAction = nil

	case spinner.TickMsg:
		if m.pendingAction != nil {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			finalCmds = append(finalCmds, cmd)
		}

	case cmds.SetFocusMsg:
		if int(msg) == m.componentId {
			m.isFocused = true
		} else {
			m.isFocused = false
		}

	case cmds.SetSelectedGroupMsg:
		m.selectedGroup = string(msg)

	case cmds.SetServicesListMsg:
		m.services = msg

	case cmds.GetRunningContainersMsg:
		if msg.Err == nil {
			m.containers = msg.Containers
		}

	// A background poll withholds GetRunningContainersMsg while stats are in
	// flight and delivers the containers here instead, so this panel has to
	// answer both or its member rows only refresh on a foreground reload.
	case cmds.GetContainerStatsMsg:
		if msg.Containers != nil {
			m.containers = msg.Containers
		}

	case tea.KeyPressMsg:
		if !m.isFocused || m.selectedGroup == "" {
			break
		}

		if action, ok := chrome.DockerActionFor(msg); ok {
			actionCmd := cmds.RequestDockerAction(action, m.selectedGroup, true)
			finalCmds = append(finalCmds, actionCmd)
		}

		switch {
		case key.Matches(msg, keys.Details.Remove):
			// Remove destroys containers, so it goes through a
			// confirmation first, unlike the other four actions.
			finalCmds = append(finalCmds, cmds.OpenConfirmModal(
				fmt.Sprintf("Remove group %q?\nThis stops and removes its containers.", m.selectedGroup),
				cmds.RequestDockerAction("remove", m.selectedGroup, true),
			))

		case key.Matches(msg, keys.Details.Logs):
			finalCmds = append(finalCmds, cmds.OpenLogsModal(m.selectedGroup, true))
		}
	}

	return m, tea.Batch(finalCmds...)
}
