package utils

import (
	"fmt"
	"os/exec"
)

// Executes `docker compose ps` scoped to the compose file in the current
// directory and returns the raw JSON output. Using `docker compose ps`
// (rather than `docker ps`) means each entry already carries the compose
// service name in its "Service" field, so callers never need to guess it
// from the container name.
//
// The output shape depends on the Docker Compose version: newer releases
// emit a single JSON array, older ones emit NDJSON (one object per line).
// ParseContainers accepts both.
func DockerComposePs() (string, error) {
	command := exec.Command("docker", "compose", "ps", "--format", "json")
	output, err := command.CombinedOutput()

	if err != nil {
		return "", fmt.Errorf("docker compose ps failed: %w: %s", err, string(output))
	}

	return string(output), nil
}
