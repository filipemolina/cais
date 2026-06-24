package model

import (
	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/apptypes"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// Handle keyboard events
	case tea.KeyPressMsg:

		switch msg.String() {

		// Quit the program on Ctrl+c or q
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	// This is execute once when the app loads and after every
	// window resize.
	case tea.WindowSizeMsg:
		h, v := appstyles.DocStyle.GetFrameSize()
		m.containers.listWidth = msg.Width - h
		m.containers.listHeight = msg.Height - v

		m.containers.runningContainers.SetSize(
			m.containers.listWidth,
			m.containers.listHeight,
		)

	case []apptypes.DockerContainer:
		containersList := []list.Item{}

		for _, container := range msg {
			containersList = append(containersList, apptypes.ContainerListItem(container))
		}

		m.containers.runningContainers = list.New(
			containersList,
			list.NewDefaultDelegate(),
			m.containers.listWidth,
			m.containers.listHeight,
		)
		m.containers.runningContainers.Title = "Running Containers:"
	}

	var cmd tea.Cmd
	m.containers.runningContainers, cmd = m.containers.runningContainers.Update(msg)
	return m, cmd
}
