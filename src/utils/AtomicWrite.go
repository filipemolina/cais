package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// ReplaceFileAtomically writes contents to fileName by way of a temporary
// file that is renamed into place, creating the file if it doesn't exist.
//
// Writing over the file directly truncates it first, so anything that fails
// after that point - a full disk, a crash, a killed process - leaves the
// user with a half-written or empty compose file and no copy of the
// original. Rename, by contrast, is atomic within a filesystem: a reader,
// including `docker compose` itself, sees either the old file or the
// complete new one.
//
// Permissions of an existing file are carried over to the replacement.
func ReplaceFileAtomically(fileName string, contents []byte) error {
	// The temporary file has to share a filesystem with its target for the
	// rename to be atomic, so it goes in the target's directory rather than
	// somewhere like /tmp, which is often a separate mount. The leading dot
	// keeps it out of the way if it ever does survive.
	temp, err := os.CreateTemp(filepath.Dir(fileName), "."+filepath.Base(fileName)+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed creating a temporary file next to %s: %w", fileName, err)
	}
	tempName := temp.Name()

	// Every failure below returns without renaming, so the temporary file is
	// always cleaned up. After a successful rename both calls are harmless
	// no-ops on a name that no longer exists.
	defer func() {
		temp.Close()
		os.Remove(tempName)
	}()

	// CreateTemp opens at 0600. Match the file being replaced so an edit
	// doesn't silently tighten permissions the user chose; fall back to the
	// usual default when creating a new file.
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(fileName); statErr == nil {
		mode = info.Mode().Perm()
	}
	if err := temp.Chmod(mode); err != nil {
		return fmt.Errorf("failed setting permissions on %s: %w", tempName, err)
	}

	if _, err := temp.Write(contents); err != nil {
		return fmt.Errorf("failed writing %s: %w", fileName, err)
	}

	// Flush before renaming: the rename can otherwise reach the disk ahead of
	// the contents, leaving the compose file empty after a crash.
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("failed flushing %s: %w", fileName, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("failed writing %s: %w", fileName, err)
	}

	if err := os.Rename(tempName, fileName); err != nil {
		return fmt.Errorf("failed replacing %s: %w", fileName, err)
	}

	return nil
}
