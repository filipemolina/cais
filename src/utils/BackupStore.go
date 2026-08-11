package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// MaxBackupsPerSource caps how many past copies the store keeps for a single
// source file. Compose files are a few KB, so 500 copies stay under a
// megabyte; the cap exists only to stop a churny file from growing the folder
// forever. Pruning keeps the newest entries and drops the oldest, on insert.
const MaxBackupsPerSource = 500

// EnsureBackupStore creates the sidecar backup folder next to the resolved
// compose file. It is idempotent: calling it again is a no-op. The folder is
// local to the stack directory, not in the config dir and not under
// ~/.local/share, so it moves with the project and stays discoverable.
func EnsureBackupStore(composeDir string) error {
	if composeDir == "" {
		compareDir := "."
		composeDir = compareDir
	}
	storeDir := filepath.Join(composeDir, ".cais", "backups")
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		return fmt.Errorf("failed creating backup store in %s: %w", composeDir, err)
	}

	// Keep the store out of `git status` noise when the parent directory is a
	// repo. This never invokes git; it only leaves a standard ignore file.
	gitignorePath := filepath.Join(composeDir, ".cais", ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("backups/*\n"), 0o644); err != nil {
		return fmt.Errorf("failed writing .cais/.gitignore in %s: %w", composeDir, err)
	}

	return nil
}

// SnapshotFile captures the pre-write state of fileName into the backup
// store, so a bad write can later be undone. It is called from inside the
// atomic write, before the file is replaced, so it only ever runs for writes
// the app has already committed to.
//
// A brand-new file has nothing to snapshot, so it is a no-op that still
// succeeds. If the file's content matches the most recent entry already in
// the store, the snapshot is skipped too (nothing new to preserve).
//
// If the backup cannot be taken, the error is returned and the write is
// refused: no write happens without a retained copy of what it replaces.
func SnapshotFile(fileName string) error {
	current, err := os.ReadFile(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			// Nothing to back up yet: a fresh .env or a bootstrap create.
			return nil
		}
		return fmt.Errorf("failed reading %s before backing it up: %w", fileName, err)
	}

	slugDir := snapshotDirFor(fileName)
	if err := os.MkdirAll(slugDir, 0o755); err != nil {
		return fmt.Errorf("failed creating backup folder for %s: %w", fileName, err)
	}

	contentHash := sha256.Sum256(current)
	sha8 := hex.EncodeToString(contentHash[:])[:8]

	// Dedup: if the newest existing entry already holds this exact content,
	// there is nothing new to preserve, so skip the write.
	if newest := newestEntry(slugDir); newest != "" {
		existingSha := sha8FromName(newest)
		if existingSha != "" && existingSha == sha8 {
			return nil
		}
	}

	timestamp := time.Now().UTC().Format("20060102T150405")
	bakName := fmt.Sprintf("%s.%s.bak", timestamp, sha8)
	bakPath := filepath.Join(slugDir, bakName)

	if err := os.WriteFile(bakPath, current, 0o644); err != nil {
		return fmt.Errorf("failed writing backup of %s: %w", fileName, err)
	}

	pruneSnapshots(slugDir)
	return nil
}

// snapshotDirFor derives the source's own backup folder from the file being
// written, so SnapshotFile never needs the compose directory passed in. It
// resolves the file's directory and a slug derived from the basename, keeping
// each source's history in its own subfolder (compose and .env never
// collide).
func snapshotDirFor(fileName string) string {
	dir := filepath.Dir(fileName)
	slug := sourceSlug(filepath.Base(fileName))
	return filepath.Join(dir, ".cais", "backups", slug)
}

// sourceSlug turns a source file's basename into a safe filesystem token. The
// compose file and .env must never collide, so dots become underscores:
// compose.yaml -> compose_yaml, .env -> _env.
func sourceSlug(base string) string {
	return strings.ReplaceAll(base, ".", "_")
}

// newestEntry returns the entry filename with the most recent UTC timestamp
// prefix, or "" if the slug directory is empty. Entries sort lexically by
// their timestamp prefix, and the newest is last.
func newestEntry(slugDir string) string {
	entries, err := os.ReadDir(slugDir)
	if err != nil {
		return ""
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".bak") {
			continue
		}
		names = append(names, name)
	}

	if len(names) == 0 {
		return ""
	}

	sort.Strings(names)
	return names[len(names)-1]
}

// sha8FromName pulls the 8-hex content hash out of a .bak filename of the
// form <utc-ts>.<sha8>.bak. It returns "" if the name does not match.
func sha8FromName(name string) string {
	if !strings.HasSuffix(name, ".bak") {
		return ""
	}
	trimmed := strings.TrimSuffix(name, ".bak")
	dot := strings.LastIndex(trimmed, ".")
	if dot < 0 {
		return ""
	}
	sha := trimmed[dot+1:]
	if len(sha) != 8 {
		return ""
	}
	return sha
}

// pruneSnapshots keeps the most recent MaxBackupsPerSource entries in
// slugDir, deleting the oldest beyond the cap.
func pruneSnapshots(slugDir string) {
	entries, err := os.ReadDir(slugDir)
	if err != nil {
		return
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".bak") {
			continue
		}
		names = append(names, name)
	}

	if len(names) <= MaxBackupsPerSource {
		return
	}

	sort.Strings(names)
	// The newest are last; discard the oldest beyond the cap.
	toRemove := names[:len(names)-MaxBackupsPerSource]
	for _, name := range toRemove {
		_ = os.Remove(filepath.Join(slugDir, name))
	}
}
