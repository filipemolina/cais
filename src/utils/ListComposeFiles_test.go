package utils

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestListComposeFiles(t *testing.T) {
	dir := t.TempDir()

	names := []string{
		"compose.yaml",
		"compose.yml",
		"docker-compose.yaml",
		"compose.prod.yaml",
		"notes.txt",    // not YAML
		"README.md",    // not YAML
		".hidden.yaml", // hidden but YAML - included, the picker can show it
	}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	// A subdirectory ending in .yaml must not be picked up as a file.
	if err := os.Mkdir(filepath.Join(dir, "conf.yaml"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	got, err := ListComposeFiles(dir)
	if err != nil {
		t.Fatalf("ListComposeFiles: %v", err)
	}

	want := []string{
		".hidden.yaml",
		"compose.prod.yaml",
		"compose.yaml",
		"compose.yml",
		"docker-compose.yaml",
	}
	if !slices.Equal(got, want) {
		t.Errorf("ListComposeFiles = %v, want %v", got, want)
	}
}

func TestListComposeFiles_EmptyDir(t *testing.T) {
	got, err := ListComposeFiles(t.TempDir())
	if err != nil {
		t.Fatalf("ListComposeFiles: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected no files, got %v", got)
	}
}

func TestListComposeFiles_MissingDir(t *testing.T) {
	_, err := ListComposeFiles(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected an error for a missing directory")
	}
}
