package model

import (
	"stack-stitcher/src/cmds"

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
		switch msg.String() {

		// Quit the program on Ctrl+c or q
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	// This is executed once when the app loads and after every
	// window resize.
	case tea.WindowSizeMsg:
		m.config.terminalWidht = msg.Width
		m.config.terminalHeight = msg.Height

	// Commands from the cmds folder
	case cmds.GetConfigMsg:
		m.config.configFileName = msg.FileName
		m.config.configProject = msg.Project
	}

	// Update nested components

	var mainMenuCmd tea.Cmd
	m.components.MainMenu, mainMenuCmd = m.components.MainMenu.Update(msg)
	finalCmds = append(finalCmds, mainMenuCmd)

	var servicesListCmd tea.Cmd
	m.components.ServicesList, servicesListCmd = m.components.ServicesList.Update(msg)
	finalCmds = append(finalCmds, servicesListCmd)

	return m, tea.Batch(finalCmds...)
}

// ALSO: Check if Go has an official Docker lib
