package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// countBak returns how many .bak entries live in dir (subfolders ignored).
func countBak(t *testing.T, dir string) int {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatalf("reading %s: %v", dir, err)
	}

	n := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".bak") {
			n++
		}
	}
	return n
}

// readSoleBak reads the single .bak in dir and fails if there is not exactly
// one.
func readSoleBak(t *testing.T, dir string) []byte {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}

	var bak string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".bak") {
			if bak != "" {
				t.Fatalf("expected exactly one .bak in %s, found more", dir)
			}
			bak = e.Name()
		}
	}
	if bak == "" {
		t.Fatalf("expected a .bak in %s, found none", dir)
	}

	got, err := os.ReadFile(filepath.Join(dir, bak))
	if err != nil {
		t.Fatalf("reading %s: %v", bak, err)
	}
	return got
}

// TestEnsureBackupStoreCreatesAndIsIdempotent covers test 1: the folder is
// created, holds a .cais/.gitignore, and a second call is a no-op.
func TestEnsureBackupStoreCreatesAndIsIdempotent(t *testing.T) {
	dir := t.TempDir()

	if err := EnsureBackupStore(dir); err != nil {
		t.Fatalf("EnsureBackupStore: %v", err)
	}

	store := filepath.Join(dir, ".cais", "backups")
	if info, err := os.Stat(store); err != nil || !info.IsDir() {
		t.Fatalf("backup store not created at %s: %v", store, err)
	}

	gitignore := filepath.Join(dir, ".cais", ".gitignore")
	body, err := os.ReadFile(gitignore)
	if err != nil {
		t.Fatalf("reading .cais/.gitignore: %v", err)
	}
	if string(body) != "backups/*\n" {
		t.Errorf(".cais/.gitignore content: got %q, want \"backups/*\\n\"", string(body))
	}

	// Idempotent: a second call must not error.
	if err := EnsureBackupStore(dir); err != nil {
		t.Fatalf("EnsureBackupStore (second call): %v", err)
	}
}

// TestSnapshotFileWritesMatchingBak covers test 2: the .bak bytes equal the
// current file, and the name carries the UTC timestamp and the SHA-8.
func TestSnapshotFileWritesMatchingBak(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")

	contents := []byte("services:\n  app:\n    image: nginx:alpine\n")
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	if err := SnapshotFile(path); err != nil {
		t.Fatalf("SnapshotFile: %v", err)
	}

	slugDir := snapshotDirFor(path)
	if countBak(t, slugDir) != 1 {
		t.Fatalf("expected 1 .bak after snapshot, got %d", countBak(t, slugDir))
	}
	if got := readSoleBak(t, slugDir); string(got) != string(contents) {
		t.Errorf("backup contents: got %q, want %q", string(got), string(contents))
	}

	entries, _ := os.ReadDir(slugDir)
	name := entries[0].Name()

	sha := sha256.Sum256(contents)
	wantSha := hex.EncodeToString(sha[:])[:8]
	if !strings.HasSuffix(name, "."+wantSha+".bak") {
		t.Errorf("backup name %q should carry sha8 %q", name, wantSha)
	}
	// The UTC timestamp that prefixes the name must parse.
	ts := strings.TrimSuffix(name, "."+wantSha+".bak")
	if _, err := time.Parse("20060102T150405", ts); err != nil {
		t.Errorf("backup timestamp prefix %q should be a parseable UTC layout: %v", ts, err)
	}
}

// TestSnapshotFileDedupsUnchangedContent covers test 3: two snapshots with no
// intervening change write nothing the second time.
func TestSnapshotFileDedupsUnchangedContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")

	if err := os.WriteFile(path, []byte("v1\n"), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	if err := SnapshotFile(path); err != nil {
		t.Fatalf("SnapshotFile #1: %v", err)
	}
	if err := SnapshotFile(path); err != nil {
		t.Fatalf("SnapshotFile #2: %v", err)
	}

	if got := countBak(t, snapshotDirFor(path)); got != 1 {
		t.Errorf("expected 1 .bak after two identical snapshots, got %d", got)
	}
}

// TestSnapshotFileKeepsOldOnChange covers test 4: a change produces a new
// entry while the old one remains.
func TestSnapshotFileKeepsOldOnChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")

	if err := os.WriteFile(path, []byte("v1\n"), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	if err := SnapshotFile(path); err != nil {
		t.Fatalf("SnapshotFile #1: %v", err)
	}

	if err := os.WriteFile(path, []byte("v2\n"), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	if err := SnapshotFile(path); err != nil {
		t.Fatalf("SnapshotFile #2: %v", err)
	}

	if got := countBak(t, snapshotDirFor(path)); got != 2 {
		t.Errorf("expected 2 .bak after a change, got %d", got)
	}
}

// TestSnapshotFileBrandNewReturnsNil covers test 5: a non-existent file is a
// no-op that succeeds and writes nothing.
func TestSnapshotFileBrandNewReturnsNil(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.yaml")

	if err := SnapshotFile(path); err != nil {
		t.Fatalf("SnapshotFile on missing file: want nil, got %v", err)
	}
	if got := countBak(t, snapshotDirFor(path)); got != 0 {
		t.Errorf("expected no .bak for a brand-new file, got %d", got)
	}
}

// TestSnapshotFileRetentionPrunesToCap covers test 6: seeding > N entries
// prunes to N, keeping the newest.
func TestSnapshotFileRetentionPrunesToCap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")

	// Keep one pre-existing entry out of the loop so the count starts at 1.
	if err := os.WriteFile(path, []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	if err := SnapshotFile(path); err != nil {
		t.Fatalf("SnapshotFile seed: %v", err)
	}

	// Write MaxBackupsPerSource more distinct contents so the folder holds
	// the cap + 1 total, forcing a prune to exactly the cap.
	for i := 0; i < MaxBackupsPerSource; i++ {
		contents := []byte("content number " + strconv.Itoa(i) + "\n")
		if err := os.WriteFile(path, contents, 0o644); err != nil {
			t.Fatalf("writing fixture %d: %v", i, err)
		}
		if err := SnapshotFile(path); err != nil {
			t.Fatalf("SnapshotFile %d: %v", i, err)
		}
	}

	if got := countBak(t, snapshotDirFor(path)); got != MaxBackupsPerSource {
		t.Errorf("expected %d .bak after pruning, got %d", MaxBackupsPerSource, got)
	}
}

// TestSnapshotFileSeparateSlugsForComposeAndEnv covers test 7: compose and
// .env get separate slug folders and never collide.
func TestSnapshotFileSeparateSlugsForComposeAndEnv(t *testing.T) {
	dir := t.TempDir()
	compose := filepath.Join(dir, "compose.yaml")
	env := filepath.Join(dir, ".env")

	if err := os.WriteFile(compose, []byte("services:\n"), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	if err := os.WriteFile(env, []byte("SECRET=1\n"), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	if err := SnapshotFile(compose); err != nil {
		t.Fatalf("SnapshotFile compose: %v", err)
	}
	if err := SnapshotFile(env); err != nil {
		t.Fatalf("SnapshotFile env: %v", err)
	}

	composeSlug := filepath.Join(dir, ".cais", "backups", "compose_yaml")
	envSlug := filepath.Join(dir, ".cais", "backups", "_env")

	if countBak(t, composeSlug) != 1 {
		t.Errorf("compose slug should have 1 .bak, got %d", countBak(t, composeSlug))
	}
	if countBak(t, envSlug) != 1 {
		t.Errorf(".env slug should have 1 .bak, got %d", countBak(t, envSlug))
	}
}

// TestSourceSlugDerivation covers test 8: .env -> _env, and dots in a compose
// name become underscores.
func TestSourceSlugDerivation(t *testing.T) {
	cases := []struct {
		base string
		want string
	}{
		{".env", "_env"},
		{"compose.yaml", "compose_yaml"},
		{"docker-compose.yml", "docker-compose_yml"},
		{"my.thing.yml", "my_thing_yml"},
	}
	for _, c := range cases {
		got := sourceSlug(c.base)
		if got != c.want {
			t.Errorf("sourceSlug(%q): got %q, want %q", c.base, got, c.want)
		}
	}
}

// TestSnapshotFileFailsClosed covers test 9: when the store cannot be created
// (here, a regular file sits where the slug folder must go), SnapshotFile
// returns an error instead of silently skipping.
func TestSnapshotFileFailsClosed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")

	// The source file must exist, or SnapshotFile would return the
	// brand-new-file no-op before reaching MkdirAll.
	if err := os.WriteFile(path, []byte("SECRET=1\n"), 0o644); err != nil {
		t.Fatalf("writing source fixture: %v", err)
	}

	// A regular file at the slug location blocks MkdirAll, forcing the
	// fail-closed path.
	slugDir := snapshotDirFor(path)
	if err := os.MkdirAll(filepath.Dir(slugDir), 0o755); err != nil {
		t.Fatalf("creating parent: %v", err)
	}
	if err := os.WriteFile(slugDir, []byte("blocked\n"), 0o644); err != nil {
		t.Fatalf("writing blocker: %v", err)
	}

	if err := SnapshotFile(path); err == nil {
		t.Fatal("SnapshotFile on a blocked store: want an error, got nil")
	}
}

// TestListBackupsReturnsNewestFirst covers test 1: a seeded slug dir returns
// its entries newest-first, with timestamp and SHA-8 parsed, and Source set
// correctly for compose and .env.
func TestListBackupsReturnsNewestFirst(t *testing.T) {
	dir := t.TempDir()
	compose := filepath.Join(dir, "compose.yaml")
	env := filepath.Join(dir, ".env")

	for _, path := range []string{compose, env} {
		if err := os.WriteFile(path, []byte(path+"\n"), 0o644); err != nil {
			t.Fatalf("writing fixture: %v", err)
		}
	}

	// Two compose snapshots and one .env snapshot, written in order so the
	// names carry distinct timestamps.
	if err := SnapshotFile(compose); err != nil {
		t.Fatalf("SnapshotFile compose #1: %v", err)
	}
	if err := os.WriteFile(compose, []byte("services:\n  app:\n    image: traefik\n"), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	if err := SnapshotFile(compose); err != nil {
		t.Fatalf("SnapshotFile compose #2: %v", err)
	}
	if err := SnapshotFile(env); err != nil {
		t.Fatalf("SnapshotFile env: %v", err)
	}

	composeBackups, err := ListBackups(compose)
	if err != nil {
		t.Fatalf("ListBackups(compose): %v", err)
	}
	if len(composeBackups) != 2 {
		t.Fatalf("compose backups: got %d, want 2", len(composeBackups))
	}
	if composeBackups[0].Timestamp.Before(composeBackups[1].Timestamp) {
		t.Error("compose backups not newest-first")
	}
	if composeBackups[0].Source != "compose" || composeBackups[1].Source != "compose" {
		t.Errorf("compose backups Source: got %q/%q, want both %q",
			composeBackups[0].Source, composeBackups[1].Source, "compose")
	}
	if composeBackups[0].SHA8 == "" {
		t.Error("compose backup missing SHA-8")
	}
	if !strings.HasSuffix(composeBackups[0].Name, composeBackups[0].SHA8+".bak") {
		t.Errorf("compose backup name %q does not carry its SHA-8 %q",
			composeBackups[0].Name, composeBackups[0].SHA8)
	}

	envBackups, err := ListBackups(env)
	if err != nil {
		t.Fatalf("ListBackups(env): %v", err)
	}
	if len(envBackups) != 1 {
		t.Fatalf("env backups: got %d, want 1", len(envBackups))
	}
	if envBackups[0].Source != ".env" {
		t.Errorf("env backup Source: got %q, want %q", envBackups[0].Source, ".env")
	}
}

// TestListBackupsNeverWrittenReturnsEmpty covers test 2: a source whose slug
// folder does not exist returns (nil, nil) - an empty slice, no error - so a
// caller does not have to invent a folder just to read nothing.
func TestListBackupsNeverWrittenReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")

	backups, err := ListBackups(path)
	if err != nil {
		t.Fatalf("ListBackups on never-written file: want nil err, got %v", err)
	}
	if len(backups) != 0 {
		t.Fatalf("ListBackups on never-written file: got %d entries, want 0", len(backups))
	}
}

// TestRestoreBackupWritesBackAndIsUndoable covers test 3: the .bak bytes are
// written back to the source, and because restore routes through
// ReplaceFileAtomically the slug gains one more entry - a backup of the
// pre-restore file - so the restore is itself undoable.
func TestRestoreBackupWritesBackAndIsUndoable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")

	preRestore := "services:\n  app:\n    image: nginx:alpine\n"
	if err := os.WriteFile(path, []byte(preRestore), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	if err := SnapshotFile(path); err != nil {
		t.Fatalf("SnapshotFile: %v", err)
	}

	// Change the file so a restore has something to revert to.
	changed := "services:\n  app:\n    image: traefik\n"
	if err := os.WriteFile(path, []byte(changed), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	if err := SnapshotFile(path); err != nil {
		t.Fatalf("SnapshotFile #2: %v", err)
	}

	backups, err := ListBackups(path)
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(backups) != 2 {
		t.Fatalf("precondition: want 2 backups, got %d", len(backups))
	}
	// Newest first: the pre-restore copy is backups[1].
	target := backups[1]

	if err := RestoreBackup(path, target.Name); err != nil {
		t.Fatalf("RestoreBackup: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading restored file: %v", err)
	}
	if string(got) != preRestore {
		t.Errorf("restored content: got %q, want %q", string(got), preRestore)
	}

	// The restore is undoable: the replaced state (changed) is still in the
	// store as the newest entry, so it can be restored back. SnapshotFile
	// dedups on content, so restoring to a copy that already exists does not
	// add a new .bak - the store keeps two entries, and the pre-restore file
	// is still recoverable from backups[0] rather than from a freshly written
	// third one.
	after, err := ListBackups(path)
	if err != nil {
		t.Fatalf("ListBackups after restore: %v", err)
	}
	if len(after) != 2 {
		t.Errorf("backups after restore: got %d, want 2 (dedup keeps the count stable)", len(after))
	}
	if after[0].SHA8 != backups[0].SHA8 {
		t.Errorf("newest backup after restore: SHA-8 got %q, want %q (the pre-restore file still recoverable)",
			after[0].SHA8, backups[0].SHA8)
	}

	// Round-trip: restoring the newest entry puts the pre-restore state back
	// on disk, proving the restore was undoable.
	if err := RestoreBackup(path, after[0].Name); err != nil {
		t.Fatalf("undo RestoreBackup: %v", err)
	}
	undone, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading undone file: %v", err)
	}
	if string(undone) != changed {
		t.Errorf("undone content: got %q, want %q", string(undone), changed)
	}
}

// TestRestoreBackupUnknownNameErrors covers test 4: a name that is not a
// .bak, or that is not present in the slug dir, is rejected rather than
// silently writing the wrong bytes.
func TestRestoreBackupUnknownNameErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")
	if err := os.WriteFile(path, []byte("v1\n"), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	if err := SnapshotFile(path); err != nil {
		t.Fatalf("SnapshotFile: %v", err)
	}

	if err := RestoreBackup(path, "not-a-bak"); err == nil {
		t.Fatal("RestoreBackup with a non-.bak name: want error, got nil")
	}

	if err := RestoreBackup(path, "20990101T000000.deadbeef.bak"); err == nil {
		t.Fatal("RestoreBackup with a missing .bak name: want error, got nil")
	}
}

// TestReplaceFileAtomicallyBacksUp is the regression guard: after a replace of
// an existing file, the store holds a copy equal to the pre-write content;
// after a first-create, the store holds nothing.
func TestReplaceFileAtomicallyBacksUp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")

	// Existing target: the backup must equal the content before the write.
	original := "services:\n  app:\n    image: nginx:alpine\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	if err := ReplaceFileAtomically(path, []byte("services:\n  other:\n    image: traefik\n")); err != nil {
		t.Fatalf("ReplaceFileAtomically: %v", err)
	}

	slugDir := snapshotDirFor(path)
	if countBak(t, slugDir) != 1 {
		t.Fatalf("expected 1 backup of an existing target, got %d", countBak(t, slugDir))
	}
	if got := readSoleBak(t, slugDir); string(got) != original {
		t.Errorf("backup of existing target: got %q, want %q", string(got), original)
	}

	// First-create: nothing exists yet, so no backup is written.
	fresh := filepath.Join(dir, "new.yaml")
	if err := ReplaceFileAtomically(fresh, []byte("services:\n")); err != nil {
		t.Fatalf("ReplaceFileAtomically (create): %v", err)
	}
	if got := countBak(t, snapshotDirFor(fresh)); got != 0 {
		t.Errorf("expected no backup for a first-create, got %d", got)
	}
}
