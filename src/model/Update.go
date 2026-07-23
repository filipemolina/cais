package model

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"stack-stitcher/src/cmds"
	"stack-stitcher/src/components"

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

	orderedProfiles := m.allProfileNames()

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

	// While a modal is open, it owns all key input exclusively - the
	// underlying panels and Tab/quit handling are frozen until it closes.
	if m.activeModal != nil {
		if _, ok := msg.(tea.KeyPressMsg); ok {
			var modalCmd tea.Cmd
			m.activeModal, modalCmd = m.activeModal.Update(msg)
			return m, modalCmd
		}
	}

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

	case cmds.OpenCreateProfileModalMsg:
		if m.config.configProject != nil {
			m.activeModal = components.ProfileNameModal(m.allProfileNames(), m.config.configProject.ServiceNames())
		}

	case cmds.OpenLogsModalMsg:
		var startCmd tea.Cmd
		m.activeModal, startCmd = components.LogsModal(
			msg.Target, msg.IsProfile,
			m.config.terminalWidht, m.config.terminalHeight,
		)
		finalCmds = append(finalCmds, startCmd)

	case cmds.OpenDeleteProfileModalMsg:
		profileName := string(msg)
		m.activeModal = components.ConfirmModal(
			fmt.Sprintf("Delete profile %q? (y/n)", profileName),
			cmds.DeleteProfile(profileName),
		)

	case cmds.CloseModalMsg:
		m.activeModal = nil
		if msg.Follow != nil {
			finalCmds = append(finalCmds, msg.Follow)
		}

	case cmds.CreateProfileMsg:
		if msg.Err != nil {
			m.lastError = msg.Err.Error()
		} else {
			m.lastError = ""
			finalCmds = append(finalCmds, cmds.GetConfig)
		}

	case cmds.DeleteProfileMsg:
		if msg.Err != nil {
			m.lastError = msg.Err.Error()
		} else {
			m.lastError = ""
			finalCmds = append(finalCmds, cmds.GetConfig)
		}
	}

	if m.activeModal != nil {
		var modalCmd tea.Cmd
		m.activeModal, modalCmd = m.activeModal.Update(msg)
		finalCmds = append(finalCmds, modalCmd)
	}

	// Update nested components
	var mainMenuCmd tea.Cmd
	m.components.MainMenu, mainMenuCmd = m.components.MainMenu.Update(msg)

	innerComponentsCmd := m.UpdateInnerComponent(m.activePage, msg)
	finalCmds = append(finalCmds, mainMenuCmd, innerComponentsCmd)

	return m, tea.Batch(finalCmds...)
}
