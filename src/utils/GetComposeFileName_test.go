package utils

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// withFiles runs body in a fresh temp directory holding the given files.
func withFiles(t *testing.T, names ...string) {
	t.Helper()

	dir := t.TempDir()
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("services: {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(dir)
}

// The winner is the first candidate in Docker's order, and the rest follow in
// that same order - the UI reports both, so the order is the contract.
func TestGetComposeFileNameReportsEveryCandidateInPriorityOrder(t *testing.T) {
	withFiles(t, "docker-compose.yml", "compose.yml", "docker-compose.yaml")

	winner, candidates, err := GetComposeFileName()
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"compose.yml", "docker-compose.yaml", "docker-compose.yml"}
	if winner != want[0] {
		t.Errorf("winner: got %q, want %q", winner, want[0])
	}
	if !slices.Equal(candidates, want) {
		t.Errorf("candidates: got %v, want %v", candidates, want)
	}
}

func TestGetComposeFileNameSingleCandidate(t *testing.T) {
	withFiles(t, "docker-compose.yml")

	winner, candidates, err := GetComposeFileName()
	if err != nil {
		t.Fatal(err)
	}

	if winner != "docker-compose.yml" || len(candidates) != 1 {
		t.Errorf("got winner %q with %v, want docker-compose.yml alone", winner, candidates)
	}
}

// A directory that happens to carry a candidate name is not a compose file.
func TestGetComposeFileNameIgnoresDirectories(t *testing.T) {
	withFiles(t)
	if err := os.Mkdir("compose.yaml", 0o755); err != nil {
		t.Fatal(err)
	}

	if _, _, err := GetComposeFileName(); !errors.Is(err, ErrNoComposeFile) {
		t.Errorf("got %v, want ErrNoComposeFile", err)
	}
}

func TestGetComposeFileNameFindsNothing(t *testing.T) {
	withFiles(t, "README.md")

	if _, _, err := GetComposeFileName(); !errors.Is(err, ErrNoComposeFile) {
		t.Errorf("got %v, want ErrNoComposeFile", err)
	}
}
