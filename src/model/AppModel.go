package model

import (
	"stack-stitcher/src/apptypes"
	"stack-stitcher/src/components"

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
	MainMenu     tea.Model
	ServicesList tea.Model
}

type AppModel struct {
	navigation       navigationModel
	config           configModel
	containers       containersModel
	components       Components
	focusedComponent string
	list             list.Model
}

func GetInitialModel() AppModel {
	items := []list.Item{
		apptypes.ListItem{ItemTitle: "Raspberry Pi’s", ItemDesc: "I have ’em all over my house"},
		apptypes.ListItem{ItemTitle: "Raspberry Pi’s", ItemDesc: "I have ’em all over my house"},
		apptypes.ListItem{ItemTitle: "Raspberry Pi’s", ItemDesc: "I have ’em all over my house"},
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
			MainMenu:     components.MainMenu(),
			ServicesList: components.ServicesList(items, 0, 0),
		},
		focusedComponent: "MainMenu",

		list: list.New(items, list.NewDefaultDelegate(), 0, 0),
	}
}
