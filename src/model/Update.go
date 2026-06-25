package model

import (
	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/apptypes"
	"stack-stitcher/src/cmds"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// This var contains all the cmds that should be executed
	// at the end. Those can come from this model or from any of the
	// nested models in m.components
	var finalCmds []tea.Cmd

	switch msg := msg.(type) {
	// Handle keyboard events
	case tea.KeyPressMsg:
		updatedComponent, componentKeyPressCmd := m.components[m.focusedComponent].Update(msg)
		m.components[m.focusedComponent] = updatedComponent
		finalCmds = append(finalCmds, componentKeyPressCmd)

		switch msg.String() {

		// Quit the program on Ctrl+c or q
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	// This is executed once when the app loads and after every
	// window resize.
	case tea.WindowSizeMsg:
		h, v := appstyles.DocStyle.GetFrameSize()
		m.containers.listWidth = msg.Width - h
		m.containers.listHeight = msg.Height - v

		m.containers.runningContainers.SetSize(
			m.containers.listWidth,
			m.containers.listHeight,
		)

		updatedMainMenu, mainMenuCmd := m.components["MainMenu"].Update(msg)
		m.components["MainMenu"] = updatedMainMenu
		finalCmds = append(finalCmds, mainMenuCmd)

	// Commands from the cmds folder
	case cmds.GetRunningContainersMsg:
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

	case cmds.GetConfigMsg:
		m.config.configFileName = msg.FileName
		m.config.configProject = msg.Project
	}

	// Fix this. The m.containers.runningContainers is being used as a UI list model.
	// It should contain the information from docker, but the actual UI list should
	// be placed somewhere else.
	updatedRunningContainers, runningConainersCmd := m.containers.runningContainers.Update(msg)
	m.containers.runningContainers = updatedRunningContainers

	finalCmds = append(finalCmds, runningConainersCmd)

	return m, tea.Batch(finalCmds...)
}

// ALSO: Check if Go has an official Docker lib
