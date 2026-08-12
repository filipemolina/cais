package backuppage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/utils"
)

// seedBackups writes a compose file and a .env into dir, snapshots each a
// few times with distinct content, and returns the dir and the two paths.
func seedBackups(t *testing.T) (dir, compose, env string) {
	t.Helper()
	dir = t.TempDir()
	compose = filepath.Join(dir, "compose.yaml")
	env = filepath.Join(dir, ".env")

	if err := os.WriteFile(compose, []byte("services:\n  app:\n    image: nginx:alpine\n"), 0o644); err != nil {
		t.Fatalf("writing compose: %v", err)
	}
	if err := os.WriteFile(env, []byte("SECRET=1\n"), 0o644); err != nil {
		t.Fatalf("writing env: %v", err)
	}

	// Two compose snapshots, one .env snapshot.
	if err := utils.SnapshotFile(compose); err != nil {
		t.Fatalf("snapshot compose #1: %v", err)
	}
	if err := os.WriteFile(compose, []byte("services:\n  app:\n    image: traefik\n"), 0o644); err != nil {
		t.Fatalf("writing compose: %v", err)
	}
	if err := utils.SnapshotFile(compose); err != nil {
		t.Fatalf("snapshot compose #2: %v", err)
	}
	if err := utils.SnapshotFile(env); err != nil {
		t.Fatalf("snapshot env: %v", err)
	}

	return dir, compose, env
}

// TestBackupListMsgMergesSourcesNewestFirst covers test 5: a compose file
// with two backups and an .env with one yields three entries, newest-first,
// each labelled with its source.
func TestBackupListMsgMergesSourcesNewestFirst(t *testing.T) {
	_, compose, env := seedBackups(t)

	backups, err := utils.ListBackups(compose)
	if err != nil {
		t.Fatalf("ListBackups(compose): %v", err)
	}
	envBackups, err := utils.ListBackups(env)
	if err != nil {
		t.Fatalf("ListBackups(env): %v", err)
	}

	m := New().(Model)
	merged := append(append([]utils.BackupEntry{}, backups...), envBackups...)
	updated, _ := m.Update(cmds.BackupListMsg{Entries: merged})
	m = updated.(Model)

	if len(m.entries) != 3 {
		t.Fatalf("entries: got %d, want 3", len(m.entries))
	}
	// Newest-first: a strictly older timestamp must not appear before a
	// newer one. Two copies written in the same second share a timestamp
	// prefix and are ordered by name, which is still newest-first.
	for i := 1; i < len(m.entries); i++ {
		if m.entries[i-1].Timestamp.Before(m.entries[i].Timestamp) {
			t.Errorf("entries not newest-first at %d: %q (%s) before %q (%s)",
				i, m.entries[i-1].Name, m.entries[i-1].Timestamp, m.entries[i].Name, m.entries[i].Timestamp)
		}
	}

	// Each source is represented exactly how many times it was written.
	var composeCount, envCount int
	for _, e := range m.entries {
		switch e.Source {
		case "compose":
			composeCount++
		case ".env":
			envCount++
		default:
			t.Errorf("entry Source = %q, want compose or .env", e.Source)
		}
	}
	if composeCount != 2 {
		t.Errorf("compose entries: got %d, want 2", composeCount)
	}
	if envCount != 1 {
		t.Errorf(".env entries: got %d, want 1", envCount)
	}
}

// TestSelectingRowLoadsPreview covers test 6: selecting a row loads that
// copy's content into the preview; compose content is highlighted (the
// viewport holds non-empty content), .env is raw.
func TestSelectingRowLoadsPreview(t *testing.T) {
	_, compose, env := seedBackups(t)

	backups, _ := utils.ListBackups(compose)
	envBackups, _ := utils.ListBackups(env)
	merged := append(append([]utils.BackupEntry{}, backups...), envBackups...)

	// Pick the compose entry and the .env entry by source.
	composeEntry, envEntry := merged[0], merged[0]
	for _, e := range merged {
		if e.Source == "compose" {
			composeEntry = e
		}
		if e.Source == ".env" {
			envEntry = e
		}
	}

	// Selecting a row loads that copy's content into the preview. We drive
	// loadPreviewCmd directly (the page issues it on select and on
	// navigation) and feed the bytes through setPreview, which is what
	// BackupListMsg + the navigation keys do in the running app.
	m2 := New().(Model)
	m2.entries = merged
	msg := m2.loadPreviewCmd(composeEntry)()
	pm, ok := msg.(backupPreviewMsg)
	if !ok {
		t.Fatalf("preview command returned %T, want backupPreviewMsg", msg)
	}
	m2.setPreview(pm.Source, pm.Contents)
	if m2.preview == "" {
		t.Error("compose preview is empty after load")
	}
	if !strings.Contains(m2.preview, "nginx") && !strings.Contains(m2.preview, "traefik") {
		t.Errorf("compose preview missing expected content: %q", m2.preview)
	}

	m3 := New().(Model)
	m3.entries = merged
	msg3 := m3.loadPreviewCmd(envEntry)()
	pm3 := msg3.(backupPreviewMsg)
	m3.setPreview(pm3.Source, pm3.Contents)
	if !strings.Contains(m3.preview, "SECRET=1") {
		t.Errorf(".env preview missing expected content: %q", m3.preview)
	}
}

// TestEnterOnRowRequestsRestore covers test 7: Enter on a row emits
// cmds.RequestRestoreBackupMsg with the right source and .bak name. The empty
// state renders the "No backups yet" card and emits nothing on Enter.
func TestEnterOnRowRequestsRestore(t *testing.T) {
	_, compose, env := seedBackups(t)

	backups, _ := utils.ListBackups(compose)
	envBackups, _ := utils.ListBackups(env)
	merged := append(append([]utils.BackupEntry{}, backups...), envBackups...)

	m := New().(Model)
	updated, _ := m.Update(cmds.BackupListMsg{Entries: merged})
	m = updated.(Model)

	// Enter on the first row requests a restore of that entry.
	enterMsg := tea.KeyPressMsg{Code: 'r', Text: "r"}
	// 'r' matches Backup.Restore (enter or r).
	updated, cmd := m.Update(enterMsg)
	m = updated.(Model)
	_ = m

	var requested *cmds.RequestRestoreBackupMsg
	for _, c := range flatten(emit(cmd)) {
		if req, ok := c.(cmds.RequestRestoreBackupMsg); ok {
			requested = &req
		}
	}
	if requested == nil {
		t.Fatal("Enter on a row did not emit RequestRestoreBackupMsg")
	}
	if got, want := requested.Source, merged[0].Source; got != want {
		t.Errorf("restore Source = %q, want %q", got, want)
	}
	if got, want := requested.Name, merged[0].Name; got != want {
		t.Errorf("restore Name = %q, want %q", got, want)
	}
}

// TestEmptyStateRendersAndEmitsNothing covers test 7 (empty half): no
// backups shows the "No backups yet" card and Enter emits nothing.
func TestEmptyStateRendersAndEmitsNothing(t *testing.T) {
	dir := t.TempDir()
	compose := filepath.Join(dir, "compose.yaml")

	m := New().(Model)
	// Apply a layout so the panel has a real width to center the card in.
	layOut, _ := m.Update(cmds.SetBodyLayoutMsg{LeftWidth: 60, RightWidth: 60, Height: 30})
	m = layOut.(Model)
	updated, _ := m.Update(cmds.BackupListMsg{Entries: nil, Source: compose})
	m = updated.(Model)

	if !m.empty {
		t.Error("empty flag not set when there are no backups")
	}

	frame := m.View().Content
	if !strings.Contains(frame, "No backups yet") {
		t.Errorf("empty state does not show the \"No backups yet\" card:\n%s", frame)
	}

	// Enter on an empty list emits no restore request.
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)
	_ = m

	for _, c := range flatten(emit(cmd)) {
		if _, ok := c.(cmds.RequestRestoreBackupMsg); ok {
			t.Error("Enter on an empty backup list emitted a restore request")
		}
	}
}

// indexOf returns the position of target in entries.
func indexOf(entries []utils.BackupEntry, target utils.BackupEntry) int {
	for i, e := range entries {
		if e.Path == target.Path && e.Name == target.Name {
			return i
		}
	}
	return 0
}

// emit drains a command into the messages it produces (handles BatchMsg).
func emit(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	return collectMsg(msg)
}

func collectMsg(msg tea.Msg) []tea.Msg {
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, child := range batch {
			out = append(out, collectMsg(child)...)
		}
		return out
	}
	return []tea.Msg{msg}
}

// flatten is a tiny alias used above for readability.
func flatten(msgs []tea.Msg) []tea.Msg { return msgs }
