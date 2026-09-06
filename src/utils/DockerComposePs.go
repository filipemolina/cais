package utils

import (
	"fmt"
)

// Executes `docker compose ps` scoped to composeFile (see ComposeFileArgs)
// and returns the raw JSON output. Using `docker compose ps` (rather than
// `docker ps`) means each entry already carries the compose service name in
// its "Service" field, so callers never need to guess it from the container
// name.
//
// The output shape depends on the Docker Compose version: newer releases
// emit a single JSON array, older ones emit NDJSON (one object per line).
// ParseContainers accepts both.
//
// --all is what makes the non-running states reachable at all. Without it
// `docker compose ps` reports only running, paused and restarting containers,
// so a created, exited or dead container is not in the output and the app
// cannot tell "stopped" from "docker gave up on it" - every one of them fell
// back to the app's own "stopped". It also gives ContainerForService a stopped
// container to describe, which is what its running-preferred-with-fallback was
// written for and never had.
func DockerComposePs(composeFile string) (string, error) {
	args := append(ComposeFileArgs(composeFile), "ps", "--all", "--format", "json")

	command := dockerCommand("docker", args...)
	output, err := command.CombinedOutput()

	if err != nil {
		return "", fmt.Errorf("docker compose ps failed: %w: %s", err, string(output))
	}

	return string(output), nil
}
