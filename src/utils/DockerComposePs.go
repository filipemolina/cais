package utils

import (
	"fmt"
	"os/exec"
)

// Executes `docker compose ps` scoped to the compose file in the current
// directory and returns the output. Using `docker compose ps` (rather than
// `docker ps`) means each entry already carries the compose service name in
// its "Service" field, so callers never need to guess it from the container
// name.
func DockerComposePs() (string, error) {
	command := exec.Command("bash", "-c", "docker compose ps --format json | jq -s")
	output, err := command.CombinedOutput()

	if err != nil {
		return "", fmt.Errorf("docker compose ps failed: %w", err)
	}

	return string(output), nil
}
