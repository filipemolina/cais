package utils

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// DockerStatsContainer holds the raw output of `docker stats --no-stream --format json`.
// Field names match the Docker CLI output exactly.
type DockerStatsContainer struct {
	Container string `json:"Container"`
	ID        string `json:"ID"`
	Name      string `json:"Name"`
	CPUPerc   string `json:"CPUPerc"`
	MemPerc   string `json:"MemPerc"`
	MemUsage  string `json:"MemUsage"`
	NetIO     string `json:"NetIO"`
	BlockIO   string `json:"BlockIO"`
	PIDs      string `json:"PIDs"`
}

// DockerStats executes `docker stats --no-stream --format json` and returns
// the parsed stats for all running containers. The output is NDJSON (one
// JSON object per line).
func DockerStats() ([]DockerStatsContainer, error) {
	command := exec.Command("docker", "stats", "--no-stream", "--format", "{{json .}}")
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker stats failed: %w: %s", err, string(output))
	}

	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return []DockerStatsContainer{}, nil
	}

	var stats []DockerStatsContainer
	decoder := json.NewDecoder(strings.NewReader(trimmed))

	for {
		var stat DockerStatsContainer
		err := decoder.Decode(&stat)
		if err != nil {
			break
		}
		stats = append(stats, stat)
	}

	return stats, nil
}
