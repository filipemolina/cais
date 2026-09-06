package utils

import (
	"os/exec"
	"strings"
	"testing"
)

// --all is what makes the non-running container states reachable.
//
// Without it `docker compose ps` reports only running, paused and restarting
// containers. A created, exited or dead container is simply absent from the
// output, so the app cannot tell an ordinary stop from one docker gave up on -
// every one of them fell back to the app's own "stopped", and the inert and
// fault tiers could never be reached at all. This was found by staging the
// states against a real docker and reading the screen, not by a failing test,
// which is why there is one now.
func TestComposePsAsksForEveryContainer(t *testing.T) {
	original := dockerCommand
	t.Cleanup(func() { dockerCommand = original })

	var got []string
	dockerCommand = func(name string, args ...string) *exec.Cmd {
		got = append([]string{name}, args...)
		// Anything that exits 0 and prints nothing: the call is the subject
		// here, not the parse.
		return exec.Command("true")
	}

	if _, err := DockerComposePs("compose.yaml"); err != nil {
		t.Fatalf("DockerComposePs: %v", err)
	}

	joined := strings.Join(got, " ")
	if !strings.Contains(joined, " --all") {
		t.Errorf("compose ps ran without --all, so stopped containers are invisible: %q", joined)
	}
	if !strings.Contains(joined, " ps") {
		t.Errorf("expected a ps invocation, got %q", joined)
	}
}
