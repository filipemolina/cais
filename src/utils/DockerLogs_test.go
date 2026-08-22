package utils

import (
	"slices"
	"strings"
	"testing"
)

// A group's log stream names its member services, for the same reason
// RunDockerCompose does: `--profile X logs -f` also tails every service with
// no profiles: key, so a group's log view carried lines from services that
// are not in it.
func TestGroupLogStreamNamesItsMembersNotTheProfile(t *testing.T) {
	args, err := dockerLogsArgs("core", true, "compose.yaml", []string{"core_a", "db"})
	if err != nil {
		t.Fatalf("dockerLogsArgs: %v", err)
	}

	want := []string{"compose", "--file", "compose.yaml", "logs", "-f", "--tail", "200", "core_a", "db"}
	if !slices.Equal(args, want) {
		t.Errorf("args = %q, want %q", args, want)
	}
	if slices.Contains(args, "--profile") {
		t.Errorf("args contain --profile: %q", args)
	}
}

// A single-service log stream names that one service, unchanged.
func TestSingleServiceLogStreamNamesTheService(t *testing.T) {
	args, err := dockerLogsArgs("web", false, "compose.yaml", nil)
	if err != nil {
		t.Fatalf("dockerLogsArgs: %v", err)
	}

	want := []string{"compose", "--file", "compose.yaml", "logs", "-f", "--tail", "200", "web"}
	if !slices.Equal(args, want) {
		t.Errorf("args = %q, want %q", args, want)
	}
}

// An empty member list is refused before any process starts: `logs -f` with
// no names would tail the default set, the same over-reach as the actions.
func TestEmptyGroupLogStreamIsRefused(t *testing.T) {
	args, err := dockerLogsArgs("core", true, "compose.yaml", nil)
	if err == nil {
		t.Fatal("expected an error for an empty member list")
	}
	if !strings.Contains(err.Error(), `group "core" has no services`) {
		t.Errorf("error = %q, want it to name the group", err)
	}
	if args != nil {
		t.Errorf("args = %q, want nil when refused", args)
	}
}
