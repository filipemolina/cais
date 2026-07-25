package model

import (
	"slices"
	"stack-stitcher/src/cmds"
	"stack-stitcher/src/components"
	"stack-stitcher/src/constants"
	"stack-stitcher/src/utils"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

type navigationModel struct {
	currentPage string
}

type configModel struct {
	configFileName string
	configProject  *types.Project
	terminalWidht  int
	terminalHeight int
	// bodyLayout is the box the body panels render into. AppModel owns it
	// (see calculateBodyLayout) and broadcasts it; components never derive
	// their own size from the terminal dimensions.
	bodyLayout cmds.SetBodyLayoutMsg
}

type containersModel struct {
	runningContainers []list.Item
	runningCount      int
}

type Components struct {
	MainMenu      tea.Model
	KeybindingBar tea.Model
}

type AppModel struct {
	navigation       navigationModel
	config           configModel
	containers       containersModel
	pages            map[string][]tea.Model
	activePage       string
	components       Components
	focusedComponent int
	lastError        string
	activeModal      tea.Model
}

func (m *AppModel) ChangeFocus(index *int) tea.Cmd {
	length := len(constants.FocusableComponents)
	var finalIdx int

	if index != nil {
		finalIdx = *index

		// This happens on shift+tab
		if finalIdx == -1 {
			if m.focusedComponent > 0 {
				m.focusedComponent--
				finalIdx = m.focusedComponent
			} else {
				m.focusedComponent = length - 1
				finalIdx = m.focusedComponent
			}
		}

		if 0 <= finalIdx && finalIdx <= length-1 {
			m.focusedComponent = finalIdx
		}
	} else {
		if m.focusedComponent < length-1 {
			m.focusedComponent++
			finalIdx = m.focusedComponent
		} else {
			m.focusedComponent = 0
			finalIdx = 0
		}
	}

	return func() tea.Msg { return cmds.SetFocusMsg(finalIdx) }
}

// allGroupNames returns every distinct group referenced by any service
// in the loaded compose project, sorted. Returns nil if no project is
// loaded yet.
func (m AppModel) allGroupNames() []string {
	if m.config.configProject == nil {
		return nil
	}

	var groups []string
	for _, service := range m.config.configProject.Services {
		groups = append(groups, service.Profiles...)
	}

	groups = utils.Deduplicate(groups)
	slices.Sort(groups)

	return groups
}

// homeStats returns the counts shown in the home page status header:
// groups (distinct Compose profiles), services (total in the project),
// and running containers.
func (m AppModel) homeStats() (groups, services, running int) {
	if m.config.configProject != nil {
		groups = len(m.allGroupNames())
		services = len(m.config.configProject.Services)
	}
	running = m.containers.runningCount
	return
}

// broadcastHomeStats returns a SetHomeStats command for the home page,
// or nil if the active page isn't Home. Call this whenever the underlying
// data changes so the status header stays in sync.
func (m AppModel) broadcastHomeStats() tea.Cmd {
	if m.activePage != "Home" {
		return nil
	}
	groups, services, running := m.homeStats()
	return cmds.SetHomeStats(groups, services, running)
}

func (m *AppModel) UpdateInnerComponent(activePage string, msg tea.Msg) tea.Cmd {
	var finalCmds []tea.Cmd

	innerComponents, ok := m.pages[activePage]

	if ok {
		for idx, _ := range innerComponents {
			var componentCmd tea.Cmd
			m.pages[activePage][idx], componentCmd = m.pages[activePage][idx].Update(msg)
			finalCmds = append(finalCmds, componentCmd)
		}
	}

	return tea.Batch(finalCmds...)
}

func GetInitialModel() AppModel {
	pages := make(map[string][]tea.Model)

	pages["Home"] = []tea.Model{
		components.GroupsList([]string{}, 0, 0),
		components.GroupDetailsPanel(),
	}

	pages["Dashboard"] = []tea.Model{
		components.ServicesList([]types.ServiceConfig{}, 0, 0),
		components.DetailsPanel(nil),
	}

	return AppModel{
		containers: containersModel{
			runningContainers: []list.Item{},
		},
		config: configModel{
			configFileName: "",
			configProject:  nil,
		},
		components: Components{
			MainMenu:      components.MainMenu(),
			KeybindingBar: components.KeybindingBar(),
		},
		pages: pages,
		// Matches the cmds.SetFocus(1) sent from Init() - keeps the Tab
		// cycle counter in sync with which component is actually focused
		// at startup, so the first Tab press doesn't appear to do nothing.
		focusedComponent: 1,
	}
}
