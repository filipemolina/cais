package utils

import (
	"fmt"
	"os/exec"
)

// RunDockerCompose runs a `docker compose` action scoped either to a single
// service or to every service tagged with a profile.
//
// Remove uses `rm -fs` rather than `down`: `down` also tears down the
// project's shared network, which would affect services outside the
// selected service/profile.
func RunDockerCompose(action string, target string, isProfile bool) error {
	subcommand, ok := map[string][]string{
		"start":   {"up", "-d"},
		"stop":    {"stop"},
		"restart": {"restart"},
		"pull":    {"pull"},
		"remove":  {"rm", "-fs"},
	}[action]

	if !ok {
		return fmt.Errorf("unknown docker compose action: %s", action)
	}

	args := []string{"compose"}

	if isProfile {
		args = append(args, "--profile", target)
		args = append(args, subcommand...)
	} else {
		args = append(args, subcommand...)
		args = append(args, target)
	}

	command := exec.Command("docker", args...)
	output, err := command.CombinedOutput()

	if err != nil {
		return fmt.Errorf("docker %s failed: %w: %s", action, err, string(output))
	}

	return nil
}
