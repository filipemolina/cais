package backupslist

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

// emit drains a command into the messages it produces (handles BatchMsg).
func emit(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	return collectMsg(cmd())
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

// selectionFrom returns the selection the list published, and whether it
// published one at all.
func selectionFrom(cmd tea.Cmd) (utils.BackupEntry, bool) {
	for _, msg := range emit(cmd) {
		if sel, ok := msg.(cmds.SetSelectedBackupMsg); ok {
			return utils.BackupEntry(sel), true
		}
	}
	return utils.BackupEntry{}, false
}

// loadedList is the version list showing three copies, cursor on the first.
func loadedList(t *testing.T) Model {
	t.Helper()

	_, compose, env := seedBackups(t)

	composeBackups, err := utils.ListBackups(compose)
	if err != nil {
		t.Fatalf("ListBackups(compose): %v", err)
	}
	envBackups, err := utils.ListBackups(env)
	if err != nil {
		t.Fatalf("ListBackups(env): %v", err)
	}

	merged := append(append([]utils.BackupEntry{}, composeBackups...), envBackups...)

	updated, _ := New().Update(cmds.BackupListMsg{Entries: merged})
	m, ok := updated.(Model)
	if !ok {
		t.Fatalf("expected a Model, got %T", updated)
	}
	if len(m.entries) < 3 {
		t.Fatalf("precondition: %d entries, want at least 3", len(m.entries))
	}
	if m.selectedIdx != 0 {
		t.Fatalf("precondition: cursor starts at %d, want 0", m.selectedIdx)
	}

	return m
}

func pressKey(t *testing.T, m Model, msg tea.KeyPressMsg) Model {
	t.Helper()

	updated, _ := m.Update(msg)
	next, ok := updated.(Model)
	if !ok {
		t.Fatalf("expected a Model, got %T", updated)
	}

	return next
}

// A compose file with two backups and an .env with one yields three entries,
// newest-first, each labelled with its source.
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

// r on a row emits cmds.RequestRestoreBackupMsg with the right source and
// .bak name.
func TestRestoreKeyOnRowRequestsRestore(t *testing.T) {
	m := loadedList(t)

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})

	var requested *cmds.RequestRestoreBackupMsg
	for _, c := range emit(cmd) {
		if req, ok := c.(cmds.RequestRestoreBackupMsg); ok {
			requested = &req
		}
	}
	if requested == nil {
		t.Fatal("r on a row did not emit RequestRestoreBackupMsg")
	}
	if got, want := requested.Source, m.entries[0].Source; got != want {
		t.Errorf("restore Source = %q, want %q", got, want)
	}
	if got, want := requested.Name, m.entries[0].Name; got != want {
		t.Errorf("restore Name = %q, want %q", got, want)
	}
}

// No backups shows the "No backups yet" card, emits no restore request, and
// clears the preview's selection rather than leaving it on a stale copy.
func TestEmptyStateRendersAndEmitsNothing(t *testing.T) {
	dir := t.TempDir()
	compose := filepath.Join(dir, "compose.yaml")

	m := New().(Model)
	// Apply a layout so the panel has a real width to center the card in.
	layOut, _ := m.Update(cmds.SetBodyLayoutMsg{LeftWidth: 60, RightWidth: 60, Height: 30})
	m = layOut.(Model)
	updated, cmd := m.Update(cmds.BackupListMsg{Entries: nil, Source: compose})
	m = updated.(Model)

	if !m.empty {
		t.Error("empty flag not set when there are no backups")
	}

	sel, published := selectionFrom(cmd)
	if !published {
		t.Error("an empty store published no selection, so the preview keeps whatever it had")
	}
	if sel.Name != "" {
		t.Errorf("an empty store published %q, want the cleared selection", sel.Name)
	}

	frame := m.View().Content
	if !strings.Contains(frame, "No backups yet") {
		t.Errorf("empty state does not show the \"No backups yet\" card:\n%s", frame)
	}

	_, cmd = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	for _, c := range emit(cmd) {
		if _, ok := c.(cmds.RequestRestoreBackupMsg); ok {
			t.Error("r on an empty backup list emitted a restore request")
		}
	}
}

// Regression test: up/down and j/k move the cursor through the copies.
//
// The navigation case used to be guarded by keys.List.Navigate, which is a
// help-only binding - a label for the footer, declared with WithHelp and no
// WithKeys at all. key.Matches against a binding with no keys is always
// false, so the whole case was unreachable and the arrows did nothing. This
// panel hand-rolls its list rather than embedding a bubbles one, so nothing
// else was moving the cursor either.
func TestTheBackupListNavigates(t *testing.T) {
	for _, tc := range []struct {
		name     string
		down, up tea.KeyPressMsg
	}{
		{"arrows", tea.KeyPressMsg{Code: tea.KeyDown}, tea.KeyPressMsg{Code: tea.KeyUp}},
		{"jk", tea.KeyPressMsg{Code: 'j', Text: "j"}, tea.KeyPressMsg{Code: 'k', Text: "k"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := loadedList(t)

			if m = pressKey(t, m, tc.down); m.selectedIdx != 1 {
				t.Fatalf("after one down the cursor is at %d, want 1", m.selectedIdx)
			}
			if m = pressKey(t, m, tc.down); m.selectedIdx != 2 {
				t.Fatalf("after two down the cursor is at %d, want 2", m.selectedIdx)
			}
			if m = pressKey(t, m, tc.up); m.selectedIdx != 1 {
				t.Fatalf("after up the cursor is at %d, want 1", m.selectedIdx)
			}
		})
	}
}

// The cursor stops at both ends rather than wrapping or running off.
func TestTheBackupListStopsAtBothEnds(t *testing.T) {
	m := loadedList(t)

	for range len(m.entries) + 3 {
		m = pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if got, want := m.selectedIdx, len(m.entries)-1; got != want {
		t.Errorf("holding down left the cursor at %d, want %d", got, want)
	}

	for range len(m.entries) + 3 {
		m = pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	}
	if m.selectedIdx != 0 {
		t.Errorf("holding up left the cursor at %d, want 0", m.selectedIdx)
	}
}

// Moving the cursor publishes the row it landed on. This is the seam the
// split introduced: the list no longer reads the copy itself, so if it stops
// publishing, the preview silently keeps showing the previous version.
func TestMovingTheCursorPublishesTheSelection(t *testing.T) {
	m := loadedList(t)

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = updated.(Model)

	sel, published := selectionFrom(cmd)
	if !published {
		t.Fatal("moving the cursor published no selection, so the preview never reloads")
	}
	if want := m.entries[m.selectedIdx]; sel.Path != want.Path {
		t.Errorf("published %q, but the cursor is on %q", sel.Path, want.Path)
	}
}

// Loading the store publishes the first row, so the preview has something to
// show without the user pressing anything.
func TestLoadingPublishesTheFirstRow(t *testing.T) {
	_, compose, _ := seedBackups(t)

	backups, err := utils.ListBackups(compose)
	if err != nil {
		t.Fatalf("ListBackups(compose): %v", err)
	}

	_, cmd := New().Update(cmds.BackupListMsg{Entries: backups})

	sel, published := selectionFrom(cmd)
	if !published {
		t.Fatal("loading the store published no selection")
	}
	if sel.Path != backups[0].Path {
		t.Errorf("published %q, want the newest copy %q", sel.Path, backups[0].Path)
	}
}

// A store that fails to read clears the selection: the panel that shows the
// error is not the panel holding the stale bytes.
func TestAReadErrorClearsTheSelection(t *testing.T) {
	m := loadedList(t)

	_, cmd := m.Update(cmds.BackupListMsg{Err: os.ErrPermission})

	sel, published := selectionFrom(cmd)
	if !published {
		t.Fatal("a failed read published no selection, so the preview keeps the last copy")
	}
	if sel.Name != "" {
		t.Errorf("a failed read published %q, want the cleared selection", sel.Name)
	}
}
