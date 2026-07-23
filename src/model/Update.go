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

// configSyncCmds re-derives the ordered services/profiles lists from the
// loaded compose project and broadcasts them. Messages only reach the
// currently active page's components (see UpdateInnerComponent), so this
// needs to run both right after the config loads AND whenever the active
// page changes - otherwise a page that wasn't active at load time (e.g.
// Dashboard, since Home is active first) would never receive its services.
func (m AppModel) configSyncCmds() []tea.Cmd {
	if m.config.configProject == nil {
		return nil
	}

	var syncCmds []tea.Cmd

	length := len(m.config.configProject.Services) + len(m.config.configProject.DisabledServices)
	orderedServices := make([]types.ServiceConfig, 0, length)

	orderedServicesMap := m.config.configProject.Services
	maps.Copy(orderedServicesMap, m.config.configProject.DisabledServices)

	for _, service := range orderedServicesMap {
		orderedServices = append(orderedServices, service)
	}

	slices.SortFunc(orderedServices, func(a, b types.ServiceConfig) int {
		return cmp.Compare(a.Name, b.Name)
	})

	syncCmds = append(syncCmds, cmds.SetServicesList(orderedServices))
	if len(orderedServices) > 0 {
		syncCmds = append(syncCmds, cmds.SetSelectedService(orderedServices[0]))
	}

	orderedProfiles := make([]string, 0, len(m.config.configProject.Profiles))

	for _, service := range m.config.configProject.Services {
		orderedProfiles = append(orderedProfiles, service.Profiles...)
	}

	orderedProfiles = utils.Deduplicate(orderedProfiles)
	slices.Sort(orderedProfiles)

	syncCmds = append(syncCmds, cmds.SetProfilesList(orderedProfiles))
	if len(orderedProfiles) > 0 {
		syncCmds = append(syncCmds, cmds.SetSelectedProfile(orderedProfiles[0]))
	}

	return syncCmds
}

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
		// Refresh container state, and re-sync services/profiles, so the
		// newly active page's components have data to show even if they
		// weren't active when it was first loaded.
		finalCmds = append(finalCmds, cmds.GetRunningContainers)
		finalCmds = append(finalCmds, m.configSyncCmds()...)

	case cmds.GetRunningContainersMsg:
		if msg.Err != nil {
			m.lastError = msg.Err.Error()
		} else {
			m.lastError = ""
		}

	case cmds.DockerActionMsg:
		if msg.Err != nil {
			m.lastError = msg.Err.Error()
		} else {
			m.lastError = ""
			finalCmds = append(finalCmds, cmds.GetRunningContainers)
		}

	case cmds.GetConfigMsg:
		if msg.Err != nil {
			m.lastError = msg.Err.Error()
			break
		}

		m.config.configFileName = msg.FileName
		m.config.configProject = msg.Project
		finalCmds = append(finalCmds, m.configSyncCmds()...)
	}

	// Update nested components
	var mainMenuCmd tea.Cmd
	m.components.MainMenu, mainMenuCmd = m.components.MainMenu.Update(msg)

	innerComponentsCmd := m.UpdateInnerComponent(m.activePage, msg)
	finalCmds = append(finalCmds, mainMenuCmd, innerComponentsCmd)

	return m, tea.Batch(finalCmds...)
}
