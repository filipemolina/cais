package model

import (
	"stack-stitcher/src/apptypes"
	"stack-stitcher/src/components"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

type navigationModel struct {
	currentPage apptypes.Page
}

type configModel struct {
	configFileName string
	configProject  *types.Project
}

type containersModel struct {
	runningContainers list.Model
	listHeight        int
	listWidth         int
}

type AppComponents map[string]tea.Model

type AppModel struct {
	navigation       navigationModel
	config           configModel
	containers       containersModel
	components       AppComponents
	focusedComponent string
}

func GetInitialModel() AppModel {
	// Initialize the list with an empty slice so it's ready for messages
	emptyItems := []list.Item{}
	runningList := list.New(emptyItems, list.NewDefaultDelegate(), 0, 0)
	runningList.Title = "Running Containers:"

	components := map[string]tea.Model{
		"MainMenu": components.MainMenu(),
	}

	return AppModel{
		containers: containersModel{
			runningContainers: runningList,
			listHeight:        0,
			listWidth:         0,
		},
		config: configModel{
			configFileName: "",
			configProject:  nil,
		},
		components:       components,
		focusedComponent: "MainMenu",
	}
}
