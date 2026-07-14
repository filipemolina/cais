package model

import (
	"cmp"
	"maps"
	"slices"
	"stack-stitcher/src/cmds"
	"stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
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

		case "tab":
			tabCmd := m.ChangeFocus(nil)
			finalCmds = append(finalCmds, tabCmd)

		case "shift+tab":
			idx := int(-1)
			tabCmd := m.ChangeFocus(&idx)
			finalCmds = append(finalCmds, tabCmd)
		}

	// This is executed once when the app loads and after every
	// window resize.
	case tea.WindowSizeMsg:
		m.config.terminalWidht = msg.Width
		m.config.terminalHeight = msg.Height

	// Commands from the cmds folder
	case cmds.SetActivePageMsg:
		m.activePage = string(msg)

	case cmds.GetConfigMsg:
		// Services
		length := len(msg.Project.Services) + len(msg.Project.DisabledServices)
		orderedServices := make([]types.ServiceConfig, 0, length)

		orderedServicesMap := msg.Project.Services
		maps.Copy(orderedServicesMap, msg.Project.DisabledServices)

		for _, service := range orderedServicesMap {
			orderedServices = append(orderedServices, service)
		}

		slices.SortFunc(orderedServices, func(a, b types.ServiceConfig) int {
			return cmp.Compare(a.Name, b.Name)
		})

		servicesListCmd := cmds.SetServicesList(orderedServices)
		finalCmds = append(finalCmds, servicesListCmd)
		if len(orderedServices) > 0 {
			selectedServiceCmd := cmds.SetSelectedService(orderedServices[0])
			finalCmds = append(finalCmds, selectedServiceCmd)
		}

		// Profiles

		orderedProfiles := make([]string, 0, len(msg.Project.Profiles))

		for _, service := range msg.Project.Services {
			profiles := service.Profiles
			orderedProfiles = append(orderedProfiles, profiles...)
		}

		orderedProfiles = utils.Deduplicate(orderedProfiles)
		slices.Sort(orderedProfiles)

		profilesListCmd := cmds.SetProfilesLit(orderedProfiles)
		finalCmds = append(finalCmds, profilesListCmd)
		if len(orderedProfiles) > 0 {
			selecteProfileCmd := cmds.SetSelectedProfile(orderedProfiles[0])
			finalCmds = append(finalCmds, selecteProfileCmd)
		}

		m.config.configFileName = msg.FileName
		m.config.configProject = msg.Project
	}

	// Update nested components
	var mainMenuCmd tea.Cmd
	m.components.MainMenu, mainMenuCmd = m.components.MainMenu.Update(msg)

	innerComponentsCmd := m.UpdateInnerComponent(m.activePage, msg)
	finalCmds = append(finalCmds, mainMenuCmd, innerComponentsCmd)

	return m, tea.Batch(finalCmds...)
}
