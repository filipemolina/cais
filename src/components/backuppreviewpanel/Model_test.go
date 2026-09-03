package backuppreviewpanel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/utils"
)

// seedBackups writes a compose file and a .env into a temp dir and snapshots
// each, so there are real .bak files on disk to read back.
func seedBackups(t *testing.T) (compose, env string) {
	t.Helper()
	dir := t.TempDir()
	compose = filepath.Join(dir, "compose.yaml")
	env = filepath.Join(dir, ".env")

	if err := os.WriteFile(compose, []byte("services:\n  app:\n    image: nginx:alpine\n"), 0o644); err != nil {
		t.Fatalf("writing compose: %v", err)
	}
	if err := os.WriteFile(env, []byte("SECRET=1\n"), 0o644); err != nil {
		t.Fatalf("writing env: %v", err)
	}
	if err := utils.SnapshotFile(compose); err != nil {
		t.Fatalf("snapshot compose: %v", err)
	}
	if err := utils.SnapshotFile(env); err != nil {
		t.Fatalf("snapshot env: %v", err)
	}

	return compose, env
}

func entryFor(t *testing.T, source string) utils.BackupEntry {
	t.Helper()

	compose, env := seedBackups(t)
	path := compose
	if source == ".env" {
		path = env
	}

	entries, err := utils.ListBackups(path)
	if err != nil {
		t.Fatalf("ListBackups(%s): %v", path, err)
	}
	if len(entries) == 0 {
		t.Fatalf("no backups seeded for %s", path)
	}
	return entries[0]
}

// selectAndLoad drives the panel the way the running app does: a selection
// arrives, the panel answers with a read, and the read's result comes back.
func selectAndLoad(t *testing.T, m Model, entry utils.BackupEntry) Model {
	t.Helper()

	updated, cmd := m.Update(cmds.SetSelectedBackupMsg(entry))
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("selecting a copy issued no read, so the panel stays blank")
	}

	updated, _ = m.Update(cmd())
	return updated.(Model)
}

// Selecting a compose copy loads its bytes. The content is what a restore
// would write, so the panel must show the file's own text.
func TestSelectingAComposeCopyLoadsIt(t *testing.T) {
	m := selectAndLoad(t, New().(Model), entryFor(t, "compose"))

	if m.content == "" {
		t.Fatal("compose preview is empty after load")
	}
	if !strings.Contains(m.content, "nginx") {
		t.Errorf("compose preview missing expected content: %q", m.content)
	}
}

// A .env copy is shown raw, secrets and all. This is the page's standing
// contract - the preview is exactly what a restore would put back - so a
// change that starts masking values breaks it here on purpose.
func TestSelectingAnEnvCopyShowsItRaw(t *testing.T) {
	m := selectAndLoad(t, New().(Model), entryFor(t, ".env"))

	if !strings.Contains(m.content, "SECRET=1") {
		t.Errorf(".env preview missing expected content: %q", m.content)
	}
}

// The cleared selection empties the panel rather than leaving the previous
// copy on screen under a header that no longer names it.
func TestClearingTheSelectionEmptiesThePanel(t *testing.T) {
	m := selectAndLoad(t, New().(Model), entryFor(t, "compose"))
	if m.content == "" {
		t.Fatal("precondition: nothing loaded to clear")
	}

	updated, _ := m.Update(cmds.SetSelectedBackupMsg(utils.BackupEntry{}))
	m = updated.(Model)

	if m.content != "" {
		t.Errorf("content survived the cleared selection: %q", m.content)
	}
	if m.hasSelection() {
		t.Error("panel still reports a selection after it was cleared")
	}

	layOut, _ := m.Update(cmds.SetBodyLayoutMsg{LeftWidth: 60, RightWidth: 60, Height: 30})
	frame := layOut.(Model).View().Content
	if !strings.Contains(frame, "Nothing selected") {
		t.Errorf("cleared panel does not show its empty state:\n%s", frame)
	}
}

// A read that lands after the cursor has moved on is dropped. Without this
// the panel would show one copy's bytes under another copy's header,
// whenever a slow read is overtaken by a faster one.
func TestAStaleReadIsDiscarded(t *testing.T) {
	current := entryFor(t, "compose")
	stale := entryFor(t, ".env")

	m := selectAndLoad(t, New().(Model), current)
	loaded := m.content

	updated, _ := m.Update(backupContentsMsg{
		Path:     stale.Path,
		Source:   stale.Source,
		Contents: []byte("SECRET=overtaken\n"),
	})
	m = updated.(Model)

	if m.content != loaded {
		t.Errorf("a read for %q overwrote the content of the selected %q", stale.Path, current.Path)
	}
}

// A copy that cannot be read reports it in place, rather than showing the
// previous copy's bytes or an empty panel with no explanation.
func TestAFailedReadIsReported(t *testing.T) {
	entry := entryFor(t, "compose")

	updated, _ := New().Update(cmds.SetSelectedBackupMsg(entry))
	m := updated.(Model)

	updated, _ = m.Update(backupContentsMsg{Path: entry.Path, Source: entry.Source, Err: os.ErrPermission})
	m = updated.(Model)

	layOut, _ := m.Update(cmds.SetBodyLayoutMsg{LeftWidth: 60, RightWidth: 60, Height: 30})
	frame := layOut.(Model).View().Content
	if !strings.Contains(frame, "Could not read this copy") {
		t.Errorf("a failed read is not reported in the panel:\n%s", frame)
	}
}

// The panel takes the right half of the body row. Taking the left half - or
// the whole row, as the merged page did - would overlap the list.
func TestThePanelSizesToTheRightHalf(t *testing.T) {
	updated, _ := New().Update(cmds.SetBodyLayoutMsg{LeftWidth: 40, RightWidth: 80, Height: 30})
	m := updated.(Model)

	if m.panelWidth != 80 {
		t.Errorf("panel width = %d, want the right half (80)", m.panelWidth)
	}
	if m.panelHeight != 30 {
		t.Errorf("panel height = %d, want 30", m.panelHeight)
	}
}

var _ tea.Model = Model{}
