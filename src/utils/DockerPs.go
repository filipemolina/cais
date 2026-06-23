package utils

import (
	"log"
	"os/exec"
)

type DockerPsMsg string

// Executes `docker ps` and returns the output.
func DockerPs() string {
	command := exec.Command("bash", "-c", "docker ps --format json | jq -s")
	output, err := command.CombinedOutput()

	if err != nil {
		log.Fatalf("Command failed: %s", err)
	}

	return string(output)
}
