package model

import (
	"stack-stitcher/src/apptypes"
	types "stack-stitcher/src/apptypes"
)

type navigationModel struct {
	currentPage types.CurrentPage
}

type configModel struct {
	configFileName string
}

type containersModel struct {
	runningContainers []apptypes.DockerContainer
}

type AppModel struct {
	navigation navigationModel
	config     configModel
	containers containersModel
}

func GetInitialModel() AppModel {
	return AppModel{}
}
