package model

import (
	"stack-stitcher/src/apptypes"
	"stack-stitcher/src/components"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

type navigationModel struct {
	currentPage apptypes.CurrentPage
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

type AppComponents struct {
	SideMenu tea.Model
}

type AppModel struct {
	navigation navigationModel
	config     configModel
	containers containersModel
	components AppComponents
}

func GetInitialModel() AppModel {
	// Initialize the list with an empty slice so it's ready for messages
	emptyItems := []list.Item{}
	runningList := list.New(emptyItems, list.NewDefaultDelegate(), 0, 0)
	runningList.Title = "Running Containers:"

	components := AppComponents{
		SideMenu: components.SideMenu(),
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
		components: components,
	}
}
