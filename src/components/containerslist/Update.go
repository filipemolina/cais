package containerslist

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/constants"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var finalCmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appstyles.DocStyle.GetFrameSize()
		totalWidth := float32(msg.Width - h)
		calculatedWidth := int(totalWidth*constants.LEFT_PANEL_WIDTH - 1)
		panelWidth := max(constants.MIN_PANEL_WIDTH, calculatedWidth)

		m.list.SetSize(
			panelWidth,
			msg.Height-v-6,
		)

	case cmds.GetRunningContainersMsg:
		containersList := []list.Item{}

		for _, container := range msg.Containers {
			containersList = append(containersList, apptypes.ContainerListItem(container))
		}

		cmd := m.list.SetItems(containersList)
		finalCmds = append(finalCmds, cmd)
	}

	// Both body panels are always active now that focus is gone, so the inner
	// list always receives the keystroke.
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	finalCmds = append(finalCmds, cmd)

	return m, tea.Batch(finalCmds...)
}
