package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/filipemolina/cais/src/utils"
)

// composeFileIn writes a compose file in a fresh temp directory and returns
// the directory and the file path.
func composeFileIn(t *testing.T) (dir string, file string) {
	t.Helper()

	dir = t.TempDir()
	file = filepath.Join(dir, "compose.yaml")
	if err := os.WriteFile(file, []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	return dir, file
}

// No flags is the ordinary run, and it must stay the ordinary run: the zero
// ComposeSource is what makes the resolver look in the current directory.
func TestParseFlagsWithoutFlags(t *testing.T) {
	source, err := parseFlags(nil)
	if err != nil {
		t.Fatal(err)
	}

	if source != (utils.ComposeSource{}) {
		t.Errorf("source: got %+v, want the zero value", source)
	}
}

func TestParseFlagsAcceptsBothSpellings(t *testing.T) {
	dir, file := composeFileIn(t)

	for _, args := range [][]string{
		{"--file", file},
		{"-f", file},
	} {
		source, err := parseFlags(args)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if source.File != file {
			t.Errorf("%v: got file %q, want %q", args, source.File, file)
		}
	}

	for _, args := range [][]string{
		{"--dir", dir},
		{"-d", dir},
	} {
		source, err := parseFlags(args)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if source.Dir != dir {
			t.Errorf("%v: got dir %q, want %q", args, source.Dir, dir)
		}
	}
}

func TestParseFlagsRejectsFileAndDirTogether(t *testing.T) {
	dir, file := composeFileIn(t)

	if _, err := parseFlags([]string{"--file", file, "--dir", dir}); err == nil {
		t.Error("expected --file with --dir to be refused")
	}
}

// A path that isn't there has to fail here, before the alternate screen is
// entered - otherwise the complaint about the command just typed appears
// inside a full-screen app the user then has to quit to read it.
func TestParseFlagsRejectsMissingPaths(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")

	if _, err := parseFlags([]string{"--file", missing}); err == nil {
		t.Error("expected a missing --file to be refused")
	}
	if _, err := parseFlags([]string{"--dir", missing}); err == nil {
		t.Error("expected a missing --dir to be refused")
	}
}

func TestParseFlagsRejectsADirectoryAsFile(t *testing.T) {
	dir, _ := composeFileIn(t)

	if _, err := parseFlags([]string{"--file", dir}); err == nil {
		t.Error("expected a directory passed to --file to be refused")
	}
}
