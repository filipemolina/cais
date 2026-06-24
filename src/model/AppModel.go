package model

import (
	types "stack-stitcher/src/apptypes"

	"charm.land/bubbles/v2/list"
)

type navigationModel struct {
	currentPage types.CurrentPage
}

type configModel struct {
	configFileName string
}

type containersModel struct {
	runningContainers list.Model
	listHeight        int
	listWidth         int
}

type AppModel struct {
	navigation navigationModel
	config     configModel
	containers containersModel
}

func GetInitialModel() AppModel {
	// Initialize the list with an empty slice so it's ready for messages
	emptyItems := []list.Item{}
	runningList := list.New(emptyItems, list.NewDefaultDelegate(), 0, 0)
	runningList.Title = "Running Containers:"

	return AppModel{
		containers: containersModel{
			runningContainers: runningList,
			listHeight:        0,
			listWidth:         0,
		},
	}
}
