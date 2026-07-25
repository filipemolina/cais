package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"stack-stitcher/src/apptypes"
)

// ParseContainers turns `docker compose ps --format json` output into a
// slice of containers. Newer Docker Compose versions emit a single JSON
// array; older ones emit NDJSON (one object per line). Both are accepted so
// the app doesn't pin a minimum compose version.
func ParseContainers(output string) ([]apptypes.DockerContainer, error) {
	trimmed := strings.TrimSpace(output)

	if trimmed == "" {
		return []apptypes.DockerContainer{}, nil
	}

	if strings.HasPrefix(trimmed, "[") {
		containers := []apptypes.DockerContainer{}
		if err := json.Unmarshal([]byte(trimmed), &containers); err != nil {
			return nil, fmt.Errorf("failed parsing docker compose ps output: %w", err)
		}

		return containers, nil
	}

	containers := []apptypes.DockerContainer{}
	decoder := json.NewDecoder(strings.NewReader(trimmed))

	for {
		var container apptypes.DockerContainer
		err := decoder.Decode(&container)

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed parsing docker compose ps output: %w", err)
		}

		containers = append(containers, container)
	}

	return containers, nil
}
