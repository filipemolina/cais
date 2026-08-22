package utils

import (
	"fmt"
	"os/exec"
)

// RunDockerCompose runs a `docker compose` action scoped either to a single
// service or to an explicit set of services (a group's members).
//
// composeFile is the file the app resolved and is showing in its UI; it is
// passed to docker as --file so the command acts on the same file the panels
// describe. Empty means "let docker resolve it", which is only correct before
// a file is loaded - see ComposeFileArgs.
//
// Remove uses `rm -fs` rather than `down`: `down` also tears down the
// project's shared network, which would affect services outside the
// selected service/profile.
// composeActionArgs builds the argument list for a docker compose action
// without running anything, so tests can assert on it. It is the whole
// decision RunDockerCompose makes; the function itself only shells out.
func composeActionArgs(action string, target string, isGroup bool, composeFile string, members []string) ([]string, error) {
	subcommand, ok := map[string][]string{
		"start":   {"up", "-d"},
		"stop":    {"stop"},
		"restart": {"restart"},
		"pull":    {"pull"},
		"remove":  {"rm", "-fs"},
	}[action]

	if !ok {
		return nil, fmt.Errorf("unknown docker compose action: %s", action)
	}

	args := ComposeFileArgs(composeFile)

	if isGroup {
		// A group action names its member services rather than requesting
		// the profile. `--profile X <verb>` also reaches every service that
		// carries no profiles: key at all, so starting one group started
		// (and stopping one group stopped) services the user never selected
		// - Compose's own semantics, and not something a flag turns off.
		// Naming the services scopes the command to exactly them, and
		// auto-enables their profile on the way.
		//
		// An empty member list is refused rather than run: `up -d` with no
		// names is "start the default set", which is the very over-reach
		// this replaced. Callers should not get this far - see AppModel's
		// RunDockerActionMsg case - so this is the backstop, not the message
		// the user reads.
		if len(members) == 0 {
			return nil, fmt.Errorf("group %q has no services to %s", target, action)
		}

		args = append(args, subcommand...)
		args = append(args, members...)
	} else {
		args = append(args, subcommand...)
		args = append(args, target)
	}

	return args, nil
}

func RunDockerCompose(action string, target string, isGroup bool, composeFile string, members []string) error {
	args, err := composeActionArgs(action, target, isGroup, composeFile, members)
	if err != nil {
		return err
	}

	command := exec.Command("docker", args...)
	output, err := command.CombinedOutput()

	if err != nil {
		return fmt.Errorf("docker %s failed: %w: %s", action, err, string(output))
	}

	return nil
}
