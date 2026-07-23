package utils

import (
	"encoding/json"
	"fmt"
	"stack-stitcher/src/apptypes"
)

func ParseContainers(jsonString string) ([]apptypes.DockerContainer, error) {
	var containers = []apptypes.DockerContainer{}

	err := json.Unmarshal([]byte(jsonString), &containers)

	if err != nil {
		return nil, fmt.Errorf("failed parsing docker compose ps output: %w", err)
	}

	return containers, nil
}
