package model

import (
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
	DetailsPanel tea.Model
}

type AppModel struct {
	navigation       navigationModel
	config           configModel
	containers       containersModel
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
	return AppModel{
		containers: containersModel{
			runningContainers: []list.Item{},
		},
		config: configModel{
			configFileName: "",
			configProject:  nil,
		},
		components: Components{
			MainMenu:       components.MainMenu(),
			ContainersList: components.ContainersList([]list.Item{}, 0, 0),
			ServicesList:   components.ServicesList([]list.Item{}, 0, 0),
			DetailsPanel:   components.DetailsPanel(nil),
		},
	}
}
