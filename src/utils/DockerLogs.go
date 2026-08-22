package utils

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
)

// logTailCount is how many past lines `logs -f` replays before following.
const logTailCount = "200"

// StreamDockerLogs starts `docker compose logs -f` for a single service or for
// every service tagged with a profile, scoped to composeFile (see
// ComposeFileArgs), and streams each output line over the returned channel. The channel is closed when the process exits (or is
// cancelled). Call the returned CancelFunc to kill the process and stop the
// stream - this is the first long-lived subprocess in the app, so tearing it
// down is the caller's responsibility.
//
// Unlike RunDockerCompose, which captures CombinedOutput() once and returns,
// this reads stdout+stderr incrementally on a goroutine so the TUI can render
// lines as they arrive.
// dockerLogsArgs builds the argument list for `docker compose logs -f`
// without starting a process, so tests can assert on it.
func dockerLogsArgs(target string, isGroup bool, composeFile string, members []string) ([]string, error) {
	args := ComposeFileArgs(composeFile)
	if isGroup {
		// Named services, for the same reason RunDockerCompose names them:
		// `--profile X logs -f` also tails every service with no profiles:
		// key, so a group's log view carried lines from services that are
		// not in it.
		if len(members) == 0 {
			return nil, fmt.Errorf("group %q has no services to tail", target)
		}
		args = append(args, "logs", "-f", "--tail", logTailCount)
		args = append(args, members...)
	} else {
		args = append(args, "logs", "-f", "--tail", logTailCount, target)
	}

	return args, nil
}

func StreamDockerLogs(target string, isGroup bool, composeFile string, members []string) (<-chan string, context.CancelFunc, error) {
	ctx, cancel := context.WithCancel(context.Background())

	args, err := dockerLogsArgs(target, isGroup, composeFile, members)
	if err != nil {
		cancel()
		return nil, nil, err
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
