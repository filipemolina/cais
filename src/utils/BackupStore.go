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

// BackupEntry describes one stored version of a source file.
type BackupEntry struct {
	// Source is "compose" for any source file whose basename is not
	// ".env", and ".env" for the .env file. It is derived from the live
	// file's basename, not from the slug folder, so a merged list can
	// label each row by what it is.
	Source string
	// Name is the .bak filename, e.g. 20260811T091530.ab12cd34.bak.
	Name string
	// Timestamp is the UTC write time parsed from the filename prefix.
	Timestamp time.Time
	// SHA8 is the content hash (8 hex) parsed from the filename.
	SHA8 string
	// Path is the absolute path to the .bak, for restore.
	Path string
}

// sourceLabel returns the user-facing source tag for a file: ".env" when
// the basename is exactly .env, otherwise "compose". The backup store keys
// on the slug, but a merged list is easier to read with a label that says
// which live file the copy came from.
func sourceLabel(fileName string) string {
	if filepath.Base(fileName) == ".env" {
		return ".env"
	}
	return "compose"
}

// ListBackups returns the stored versions of sourceFile, newest first. A
// source whose slug folder does not exist - never written, or brand-new -
// yields an empty, non-error slice, so a reader need not special-case a
// missing directory. The timestamp and SHA-8 come from the filename, which
// SnapshotFile wrote, so no file is opened to learn them.
func ListBackups(sourceFile string) ([]BackupEntry, error) {
	slugDir := snapshotDirFor(sourceFile)

	entries, err := os.ReadDir(slugDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed reading backup folder for %s: %w", sourceFile, err)
	}

	label := sourceLabel(sourceFile)
	var backups []BackupEntry
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".bak") {
			continue
		}
		sha := sha8FromName(name)
		if sha == "" {
			continue
		}

		tsPart := strings.TrimSuffix(name, "."+sha+".bak")
		ts, terr := time.Parse("20060102T150405", tsPart)
		if terr != nil {
			// A malformed name is not a valid backup; skip rather than
			// invent a zero timestamp that would sort unpredictably.
			continue
		}

		backups = append(backups, BackupEntry{
			Source:    label,
			Name:      name,
			Timestamp: ts,
			SHA8:      sha,
			Path:      filepath.Join(slugDir, name),
		})
	}

	// Names sort lexically by their UTC prefix, so the newest is last;
	// reverse to newest-first.
	sort.SliceStable(backups, func(i, j int) bool {
		return backups[i].Name > backups[j].Name
	})

	return backups, nil
}

// RestoreBackup writes the named .bak back over sourceFile. It goes through
// ReplaceFileAtomically, so the file being overwritten is snapshotted first
// and the restore is itself undoable: restoring compose.yaml@T2 leaves a new
// backup of the current compose.yaml (the one about to be replaced) as the
// next-newest entry.
//
// The .bak name must belong to sourceFile's slug dir; an unknown or mismatched
// name is an error so a caller cannot accidentally write bytes from another
// source's folder over this one.
func RestoreBackup(sourceFile, backupName string) error {
	slugDir := snapshotDirFor(sourceFile)

	if sha8FromName(backupName) == "" {
		return fmt.Errorf("backup name %q is not a valid .bak", backupName)
	}

	bakPath := filepath.Join(slugDir, backupName)
	if _, err := os.Stat(bakPath); err != nil {
		return fmt.Errorf("backup %q not found for %s: %w", backupName, sourceFile, err)
	}

	contents, err := os.ReadFile(bakPath)
	if err != nil {
		return fmt.Errorf("failed reading backup %q: %w", backupName, err)
	}

	// ReplaceFileAtomically snapshots the current file before replacing it,
	// which is the undo for this restore.
	if err := ReplaceFileAtomically(sourceFile, contents); err != nil {
		return fmt.Errorf("failed restoring %s from %q: %w", sourceFile, backupName, err)
	}

	return nil
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
