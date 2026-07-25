package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tempFileCount reports how many of the helper's temporary files are left in
// dir. It should be zero after every call, successful or not: the project
// directory is the user's own, and stray .compose.yaml.tmp-* files there
// would be both confusing and picked up by shell globs.
func tempFileCount(t *testing.T, dir string) int {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}

	count := 0
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp-") {
			count++
		}
	}

	return count
}

func TestReplaceFileAtomicallyOverwritesExistingContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")

	if err := os.WriteFile(path, []byte("old contents\n"), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	if err := ReplaceFileAtomically(path, []byte("new contents\n")); err != nil {
		t.Fatalf("ReplaceFileAtomically: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading result: %v", err)
	}
	if want := "new contents\n"; string(got) != want {
		t.Errorf("contents: got %q, want %q", got, want)
	}

	if left := tempFileCount(t, dir); left != 0 {
		t.Errorf("temporary files left behind: %d, want 0", left)
	}
}

func TestReplaceFileAtomicallyCreatesMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")

	if err := ReplaceFileAtomically(path, []byte("services:\n")); err != nil {
		t.Fatalf("ReplaceFileAtomically: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat result: %v", err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o644); got != want {
		t.Errorf("new file mode: got %o, want %o", got, want)
	}
}

// An edit must not quietly tighten a compose file the user deliberately
// restricted - CreateTemp opens at 0600, so this only holds if the helper
// copies the mode across.
func TestReplaceFileAtomicallyPreservesPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")

	if err := os.WriteFile(path, []byte("old\n"), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	if err := ReplaceFileAtomically(path, []byte("new\n")); err != nil {
		t.Fatalf("ReplaceFileAtomically: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat result: %v", err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o600); got != want {
		t.Errorf("mode after replace: got %o, want %o", got, want)
	}
}

// The whole point of the temp-file dance: when the write cannot go through,
// the original file is still the original file rather than an empty one.
func TestReplaceFileAtomicallyLeavesOriginalOnFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: a read-only directory would still be writable")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")
	original := "services:\n  app:\n    image: nginx:alpine\n"

	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	// Read-only directory: the file itself stays writable, so a plain
	// truncating write would succeed and destroy it, while creating the
	// temporary file next to it fails.
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod %s: %v", dir, err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) })

	if err := ReplaceFileAtomically(path, []byte("truncated\n")); err == nil {
		t.Fatal("ReplaceFileAtomically succeeded in a read-only directory, want an error")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading result: %v", err)
	}
	if string(got) != original {
		t.Errorf("original file was modified: got %q, want %q", got, original)
	}
}
