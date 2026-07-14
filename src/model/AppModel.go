package model

import (
	"stack-stitcher/src/apptypes"
	"stack-stitcher/src/cmds"
	"stack-stitcher/src/components"
	"stack-stitcher/src/constants"

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
}

type containersModel struct {
	runningContainers []list.Item
}

type Components struct {
	MainMenu,
	ContainersList,
	ServicesList,
	DetailsPanel,
	ProfilesList tea.Model
}

type AppModel struct {
	navigation       navigationModel
	config           configModel
	containers       containersModel
	pages            map[string][]tea.Model
	components       Components
	focusedComponent int
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

func GetInitialModel() AppModel {
	pages := make(map[string][]tea.Model)

	pages["Home"] = []tea.Model{
		components.ProfilesList([]string{}, 0, 0),
		components.GroupDetailsPanel(nil),
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
		pages: pages,
		components: Components{
			MainMenu:       components.MainMenu(),
			ContainersList: components.ContainersList([]apptypes.ContainerListItem{}, 0, 0),
			ServicesList:   components.ServicesList([]types.ServiceConfig{}, 0, 0),
			DetailsPanel:   components.DetailsPanel(nil),
			ProfilesList:   components.ProfilesList([]string{}, 0, 0),
		},
	}
}
