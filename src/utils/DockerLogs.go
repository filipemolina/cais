package utils

import (
	"bufio"
	"context"
	"os/exec"
)

// logTailCount is how many past lines `logs -f` replays before following.
const logTailCount = "200"

// StreamDockerLogs starts `docker compose logs -f` for a single service or for
// every service tagged with a profile, and streams each output line over the
// returned channel. The channel is closed when the process exits (or is
// cancelled). Call the returned CancelFunc to kill the process and stop the
// stream - this is the first long-lived subprocess in the app, so tearing it
// down is the caller's responsibility.
//
// Unlike RunDockerCompose, which captures CombinedOutput() once and returns,
// this reads stdout+stderr incrementally on a goroutine so the TUI can render
// lines as they arrive.
func StreamDockerLogs(target string, isProfile bool) (<-chan string, context.CancelFunc, error) {
	ctx, cancel := context.WithCancel(context.Background())

	args := []string{"compose"}
	if isProfile {
		// Follows the same --profile convention as RunDockerCompose. Note this
		// also activates no-profile default services, so their lines can appear
		// too - identical to how start/stop already scope with --profile.
		args = append(args, "--profile", target, "logs", "-f", "--tail", logTailCount)
	} else {
		args = append(args, "logs", "-f", "--tail", logTailCount, target)
	}

	command := exec.CommandContext(ctx, "docker", args...)

	stdout, err := command.StdoutPipe()
	if err != nil {
		cancel()
		return nil, nil, err
	}
	// Merge stderr into the same pipe so docker's own error/status lines show
	// up inline in the log view rather than being lost.
	command.Stderr = command.Stdout

	if err := command.Start(); err != nil {
		cancel()
		return nil, nil, err
	}

	lines := make(chan string)

	go func() {
		defer close(lines)

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case lines <- scanner.Text():
			case <-ctx.Done():
				return
			}
		}

		// Reap the process so cancelling (or a natural exit) doesn't leave a
		// zombie behind.
		_ = command.Wait()
	}()

	return lines, cancel, nil
}
