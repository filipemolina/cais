package model

import types "stack-stitcher/src/types"

type container struct {
	containerName string
	state         string
	status        string
}

type navigationModel struct {
	currentPage types.CurrentPage
}

type configModel struct {
	configFileName string
}

type containersModel struct {
	runningContainers container
}

type AppModel struct {
	navigation navigationModel
	config     configModel
	containers containersModel
}

func GetInitialModel() AppModel {
	return AppModel{}
}
