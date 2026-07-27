package utils

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

// filesIn fills a fresh temp directory and returns it, without changing the
// working directory - the point of --dir is that the app never chdirs.
func filesIn(t *testing.T, names ...string) string {
	t.Helper()

	dir := t.TempDir()
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("services: {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return dir
}

// The winner is the first candidate in Docker's order, and the rest follow in
// that same order - the UI reports both, so the order is the contract.
func TestGetComposeFileNameReportsEveryCandidateInPriorityOrder(t *testing.T) {
	withFiles(t, "docker-compose.yml", "compose.yml", "docker-compose.yaml")

	winner, candidates, err := GetComposeFileName(ComposeSource{})
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

	winner, candidates, err := GetComposeFileName(ComposeSource{})
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

	if _, _, err := GetComposeFileName(ComposeSource{}); !errors.Is(err, ErrNoComposeFile) {
		t.Errorf("got %v, want ErrNoComposeFile", err)
	}
}

func TestGetComposeFileNameFindsNothing(t *testing.T) {
	withFiles(t, "README.md")

	if _, _, err := GetComposeFileName(ComposeSource{}); !errors.Is(err, ErrNoComposeFile) {
		t.Errorf("got %v, want ErrNoComposeFile", err)
	}
}

// --file is the answer, not a hint: the same-named file sitting in the
// current directory does not get a vote, and there is nothing to be second.
func TestGetComposeFileNameTakesAnExplicitFileAsIs(t *testing.T) {
	withFiles(t, "compose.yaml")
	elsewhere := filepath.Join(filesIn(t, "compose.yaml"), "compose.yaml")

	winner, candidates, err := GetComposeFileName(ComposeSource{File: elsewhere})
	if err != nil {
		t.Fatal(err)
	}

	if winner != elsewhere {
		t.Errorf("winner: got %q, want %q", winner, elsewhere)
	}
	if !slices.Equal(candidates, []string{elsewhere}) {
		t.Errorf("candidates: got %v, want just the named file", candidates)
	}
}

// --dir resolves the same way a bare run does, one directory over. The paths
// come back joined, because they are about to be handed to docker and to the
// YAML writers from a working directory that is somewhere else entirely.
func TestGetComposeFileNameResolvesInsideDir(t *testing.T) {
	withFiles(t)
	dir := filesIn(t, "docker-compose.yaml", "compose.yml")

	winner, candidates, err := GetComposeFileName(ComposeSource{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{filepath.Join(dir, "compose.yml"), filepath.Join(dir, "docker-compose.yaml")}
	if winner != want[0] {
		t.Errorf("winner: got %q, want %q", winner, want[0])
	}
	if !slices.Equal(candidates, want) {
		t.Errorf("candidates: got %v, want %v", candidates, want)
	}
}

// An empty --dir must not turn "compose.yaml" into "./compose.yaml": that
// string is what the footer shows and what docker is handed.
func TestGetComposeFileNameKeepsBareNamesForTheCurrentDirectory(t *testing.T) {
	withFiles(t, "compose.yaml")

	winner, _, err := GetComposeFileName(ComposeSource{})
	if err != nil {
		t.Fatal(err)
	}

	if winner != "compose.yaml" {
		t.Errorf("winner: got %q, want an unprefixed compose.yaml", winner)
	}
}

// The bootstrap flow keys off ErrNoComposeFile to offer to create one, so an
// empty --dir directory has to report it the same way an empty cwd does -
// while still saying which directory came up empty.
func TestGetComposeFileNameFindsNothingInDir(t *testing.T) {
	withFiles(t, "compose.yaml")
	dir := filesIn(t, "README.md")

	_, _, err := GetComposeFileName(ComposeSource{Dir: dir})
	if !errors.Is(err, ErrNoComposeFile) {
		t.Fatalf("got %v, want ErrNoComposeFile", err)
	}
	if got := err.Error(); !strings.Contains(got, dir) {
		t.Errorf("error %q does not name the directory %q", got, dir)
	}
}
