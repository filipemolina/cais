package model

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"slices"
	"stack-stitcher/src/apptypes"
	"stack-stitcher/src/cmds"
	"stack-stitcher/src/components"
	"stack-stitcher/src/constants"
	"stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

// calculateBodyLayout returns the exact box each body panel must render
// into: the row count left after the nav, keybinding bar, and optional error
// banner, split across the two panels so that
// left + BODY_GUTTER_WIDTH + right == the terminal width.
//
// The left panel gets LEFT_PANEL_WIDTH of the row (after the gutter is taken
// out) and the right panel gets whatever is left, so rounding can never make
// the two panels overflow or leave a ragged column. Both panels are held at
// MIN_PANEL_WIDTH where the terminal allows it; below that the row is split
// evenly and the panels clip their own content.
func (m AppModel) calculateBodyLayout() cmds.SetBodyLayoutMsg {
	menuHeight := lipgloss.Height(m.components.MainMenu.View().Content)
	keybarHeight := lipgloss.Height(m.components.KeybindingBar.View().Content)

	errorBanner := 0
	if m.lastError != "" {
		errorBanner = 1
	}

	height := m.config.terminalHeight - menuHeight - keybarHeight - errorBanner
	if height < 0 {
		height = 0
	}

	available := m.config.terminalWidht - constants.BODY_GUTTER_WIDTH
	if available < 0 {
		available = 0
	}

	var left int
	switch {
	case available < 2*constants.MIN_PANEL_WIDTH:
		left = available / 2
	default:
		left = int(float32(available) * constants.LEFT_PANEL_WIDTH)
		left = max(left, constants.MIN_PANEL_WIDTH)
		left = min(left, available-constants.MIN_PANEL_WIDTH)
	}

	return cmds.SetBodyLayoutMsg{
		LeftWidth:  left,
		RightWidth: available - left,
		Height:     height,
	}
}

// broadcastBodyLayout returns a command that sends the current body layout
// to the active page's components.
func (m AppModel) broadcastBodyLayout() tea.Cmd {
	return cmds.SetBodyLayout(
		m.config.bodyLayout.LeftWidth,
		m.config.bodyLayout.RightWidth,
		m.config.bodyLayout.Height,
	)
}

// rebroadcastBodyLayoutIfChanged recalculates the body layout and, if it
// differs from the stored value, updates the stored value and returns a
// command to broadcast it. It is used when the error banner appears or
// disappears, because the banner consumes one row.
func (m *AppModel) rebroadcastBodyLayoutIfChanged() tea.Cmd {
	newLayout := m.calculateBodyLayout()
	if newLayout == m.config.bodyLayout {
		return nil
	}
	m.config.bodyLayout = newLayout
	return m.broadcastBodyLayout()
}

// configSyncCmds re-derives the ordered services/groups lists from the
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

	orderedGroups := m.allGroupNames()

	syncCmds = append(syncCmds, cmds.SetGroupsList(orderedGroups))
	if len(orderedGroups) > 0 {
		syncCmds = append(syncCmds, cmds.SetSelectedGroup(orderedGroups[0]))
	}

	return syncCmds
}

// pageForKey returns the page an alt+<letter> chord jumps to, or "" if the key
// is not a page shortcut.
//
// It matches on the modifier field rather than on msg.String(), because
// String() returns the printable text for a key ("g") and only falls back to
// the keystroke form ("alt+g") when there is none. Requiring Mod to be exactly
// ModAlt also means ctrl+alt+g and alt+shift+g are left alone.
func pageForKey(msg tea.KeyPressMsg) string {
	key := msg.Key()

	if key.Mod != tea.ModAlt {
		return ""
	}

	return apptypes.PageForShortcut(string(key.Code))
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
		// alt+<letter> jumps straight to a page. Handled here rather than in
		// MainMenu because the nav is not focusable, so it never sees keys.
		//
		// alt, not ctrl: ctrl+s is eaten by terminal flow control (XOFF) and
		// ctrl+d is EOF, so ctrl chords on those letters are unreliable at
		// best. This runs after the modal check above, so typing in a text
		// input cannot navigate away.
		if page := pageForKey(msg); page != "" {
			if page != m.activePage {
				finalCmds = append(finalCmds, cmds.SetActivePage(page))
			}
			break
		}

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
		m.config.bodyLayout = m.calculateBodyLayout()
		finalCmds = append(finalCmds, m.broadcastBodyLayout())

	// Commands from the cmds folder
	case cmds.SetActivePageMsg:
		m.activePage = string(msg)
		// Refresh container state, and re-sync services/groups, so the
		// newly active page's components have data to show even if they
		// weren't active when it was first loaded.
		finalCmds = append(finalCmds, cmds.GetRunningContainers)
		finalCmds = append(finalCmds, m.configSyncCmds()...)
		if homeStatsCmd := m.broadcastHomeStats(); homeStatsCmd != nil {
			finalCmds = append(finalCmds, homeStatsCmd)
		}
		finalCmds = append(finalCmds, m.broadcastBodyLayout())

	case cmds.GetRunningContainersMsg:
		if msg.Err != nil {
			m.lastError = msg.Err.Error()
		} else {
			m.lastError = ""
			count := 0
			for _, container := range msg.Containers {
				if container.State == "running" {
					count++
				}
			}
			m.containers.runningCount = count
			if homeStatsCmd := m.broadcastHomeStats(); homeStatsCmd != nil {
				finalCmds = append(finalCmds, homeStatsCmd)
			}
		}
		if bodyCmd := m.rebroadcastBodyLayoutIfChanged(); bodyCmd != nil {
			finalCmds = append(finalCmds, bodyCmd)
		}

	case cmds.DockerActionMsg:
		if msg.Err != nil {
			m.lastError = msg.Err.Error()
		} else {
			m.lastError = ""
			finalCmds = append(finalCmds, cmds.GetRunningContainers)
		}
		if bodyCmd := m.rebroadcastBodyLayoutIfChanged(); bodyCmd != nil {
			finalCmds = append(finalCmds, bodyCmd)
		}

	case cmds.GetConfigMsg:
		if msg.Err != nil {
			m.lastError = msg.Err.Error()
			// No compose file in the current directory: offer to create
			// one in place. The error banner is still set above, so an
			// Esc from the modal leaves a visible explanation.
			if errors.Is(msg.Err, utils.ErrNoComposeFile) {
				m.activeModal = components.CreateComposeFileModal()
			}
			if bodyCmd := m.rebroadcastBodyLayoutIfChanged(); bodyCmd != nil {
				finalCmds = append(finalCmds, bodyCmd)
			}
			break
		}

		m.config.configFileName = msg.FileName
		m.config.configProject = msg.Project
		finalCmds = append(finalCmds, m.configSyncCmds()...)
		if homeStatsCmd := m.broadcastHomeStats(); homeStatsCmd != nil {
			finalCmds = append(finalCmds, homeStatsCmd)
		}
		if bodyCmd := m.rebroadcastBodyLayoutIfChanged(); bodyCmd != nil {
			finalCmds = append(finalCmds, bodyCmd)
		}

	case cmds.OpenCreateGroupModalMsg:
		if m.config.configProject != nil {
			m.activeModal = components.GroupNameModal(m.allGroupNames(), m.config.configProject.ServiceNames())
		}

	case cmds.OpenLogsModalMsg:
		var startCmd tea.Cmd
		m.activeModal, startCmd = components.LogsModal(
			msg.Target, msg.IsGroup,
			m.config.terminalWidht, m.config.terminalHeight,
		)
		finalCmds = append(finalCmds, startCmd)

	case cmds.OpenDeleteGroupModalMsg:
		groupName := string(msg)
		m.activeModal = components.ConfirmModal(
			fmt.Sprintf("Delete group %q? (y/n)", groupName),
			cmds.DeleteGroup(groupName),
		)

	case cmds.CloseModalMsg:
		m.activeModal = nil
		if msg.Follow != nil {
			finalCmds = append(finalCmds, msg.Follow)
		}

	case cmds.CreateGroupMsg:
		if msg.Err != nil {
			m.lastError = msg.Err.Error()
		} else {
			m.lastError = ""
			finalCmds = append(finalCmds, cmds.GetConfig)
		}
		if bodyCmd := m.rebroadcastBodyLayoutIfChanged(); bodyCmd != nil {
			finalCmds = append(finalCmds, bodyCmd)
		}

	case cmds.DeleteGroupMsg:
		if msg.Err != nil {
			m.lastError = msg.Err.Error()
		} else {
			m.lastError = ""
			finalCmds = append(finalCmds, cmds.GetConfig)
		}
		if bodyCmd := m.rebroadcastBodyLayoutIfChanged(); bodyCmd != nil {
			finalCmds = append(finalCmds, bodyCmd)
		}

	case cmds.CreateComposeFileMsg:
		if msg.Err != nil {
			m.lastError = msg.Err.Error()
		} else {
			m.lastError = ""
			finalCmds = append(finalCmds, cmds.GetConfig)
		}
		if bodyCmd := m.rebroadcastBodyLayoutIfChanged(); bodyCmd != nil {
			finalCmds = append(finalCmds, bodyCmd)
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

	var keybindingBarCmd tea.Cmd
	m.components.KeybindingBar, keybindingBarCmd = m.components.KeybindingBar.Update(msg)

	innerComponentsCmd := m.UpdateInnerComponent(m.activePage, msg)
	finalCmds = append(finalCmds, mainMenuCmd, keybindingBarCmd, innerComponentsCmd)

	return m, tea.Batch(finalCmds...)
}
