package utils

import (
	"slices"
	"testing"
)

func TestComposeFileArgsPinsTheFile(t *testing.T) {
	got := ComposeFileArgs("docker-compose.yml")
	want := []string{"compose", "--file", "docker-compose.yml"}

	if !slices.Equal(got, want) {
		t.Errorf("args: got %v, want %v", got, want)
	}
}

// --file is a compose-level flag, so it has to sit between `compose` and the
// subcommand. Appending a subcommand to the result must produce a command
// docker accepts, not `docker compose up --file X`.
func TestComposeFileArgsLeavesRoomForTheSubcommand(t *testing.T) {
	got := append(ComposeFileArgs("compose.yaml"), "up", "-d")
	want := []string{"compose", "--file", "compose.yaml", "up", "-d"}

	if !slices.Equal(got, want) {
		t.Errorf("args: got %v, want %v", got, want)
	}
}

// Before a file is loaded there is nothing to pin to, and no panel claiming
// otherwise: docker resolves it, exactly as it did before --file was threaded
// through.
func TestComposeFileArgsOmitsAnEmptyFile(t *testing.T) {
	got := ComposeFileArgs("")
	want := []string{"compose"}

	if !slices.Equal(got, want) {
		t.Errorf("args: got %v, want %v", got, want)
	}
}
