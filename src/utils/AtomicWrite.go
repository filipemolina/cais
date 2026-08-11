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
	return ReplaceFileAtomicallyWithMode(fileName, contents, 0o644)
}

// ReplaceFileAtomicallyWithMode is like ReplaceFileAtomically but accepts an
// explicit mode for new files. The mode is applied before the write, not after
// the rename, so a mode like 0600 is never briefly world-readable.
//
// If the file exists, its mode is preserved (resolving symlinks first).
// If the file is a symlink, it is resolved and the target is written through.
// If the symlink is dangling, an error is returned without creating a file.
func ReplaceFileAtomicallyWithMode(fileName string, contents []byte, newFileMode os.FileMode) error {
	// Snapshot the current file before anything touches it. This is the
	// single chokepoint every compose and .env write passes through, so one
	// insertion here captures every surface. If the snapshot cannot be taken,
	// refuse the write: no write happens without a retained copy of what it
	// replaces (a brand-new file has nothing to snapshot, and SnapshotFile
	// returns nil for that). Validation already gates the write, so a refused
	// edit never reaches this point.
	if err := SnapshotFile(fileName); err != nil {
		return fmt.Errorf("failed to back up %s before writing: %w", fileName, err)
	}

	// Resolve symlinks before writing. The temp file is created next to the
	// resolved target (D5.1), so resolving moves the write into the target's
	// directory. A read-only secrets directory now fails at CreateTemp with the
	// resolved path in the error, so the user can tell which file could not be
	// written.
	resolvedPath := fileName
	if fileInfo, err := os.Lstat(fileName); err == nil && (fileInfo.Mode()&os.ModeSymlink) != 0 {
		var symlinkErr error
		resolvedPath, symlinkErr = filepath.EvalSymlinks(fileName)
		if symlinkErr != nil {
			// Dangling symlink: report the error and don't create a regular
			// file at the symlink's path, which would silently detach the
			// user's dotfiles setup.
			return fmt.Errorf("symlink %s is dangling or broken: %w", fileName, symlinkErr)
		}
	}

	// The temporary file has to share a filesystem with its target for the
	// rename to be atomic, so it goes in the target's directory rather than
	// somewhere like /tmp, which is often a separate mount. The leading dot
	// keeps it out of the way if it ever does survive.
	temp, err := os.CreateTemp(filepath.Dir(resolvedPath), "."+filepath.Base(resolvedPath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed creating a temporary file next to %s: %w", resolvedPath, err)
	}
	tempName := temp.Name()

	// Every failure below returns without renaming, so the temporary file is
	// always cleaned up. After a successful rename both calls are harmless
	// no-ops on a name that no longer exists.
	defer func() {
		temp.Close()
		os.Remove(tempName)
	}()

	// Determine the mode: preserve existing file's mode, or use the provided
	// mode for new files. CreateTemp opens at 0600, so we need to chmod.
	// The mode must be set BEFORE the write, not after the rename, so a mode
	// like 0600 is never briefly world-readable (D5.3).
	mode := newFileMode
	if info, statErr := os.Stat(resolvedPath); statErr == nil {
		// File exists (possibly after symlink resolution); preserve its mode.
		mode = info.Mode().Perm()
	}
	if err := temp.Chmod(mode); err != nil {
		return fmt.Errorf("failed setting permissions on %s: %w", tempName, err)
	}

	if _, err := temp.Write(contents); err != nil {
		return fmt.Errorf("failed writing %s: %w", resolvedPath, err)
	}

	// Flush before renaming: the rename can otherwise reach the disk ahead of
	// the contents, leaving the file empty after a crash.
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("failed flushing %s: %w", resolvedPath, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("failed writing %s: %w", resolvedPath, err)
	}

	if err := os.Rename(tempName, resolvedPath); err != nil {
		return fmt.Errorf("failed replacing %s: %w", resolvedPath, err)
	}

	return nil
}
