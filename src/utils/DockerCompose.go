package utils

import (
	"fmt"
	"os/exec"
)

// RunDockerCompose runs a `docker compose` action scoped either to a single
// service or to every service tagged with a profile.
//
// composeFile is the file the app resolved and is showing in its UI; it is
// passed to docker as --file so the command acts on the same file the panels
// describe. Empty means "let docker resolve it", which is only correct before
// a file is loaded - see ComposeFileArgs.
//
// Remove uses `rm -fs` rather than `down`: `down` also tears down the
// project's shared network, which would affect services outside the
// selected service/profile.
func RunDockerCompose(action string, target string, isGroup bool, composeFile string) error {
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

	args := ComposeFileArgs(composeFile)

	if isGroup {
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
