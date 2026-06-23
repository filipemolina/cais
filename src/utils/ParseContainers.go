package utils

import (
	"encoding/json"
	"log"
	"stack-stitcher/src/apptypes"
)

func ParseContainers(jsonString string) []apptypes.DockerContainer {
	var containers = []apptypes.DockerContainer{}

	err := json.Unmarshal([]byte(jsonString), &containers)

	if err != nil {
		log.Fatalf("Failed parsing JSON outtup %s", err)
	}

	return containers
}
