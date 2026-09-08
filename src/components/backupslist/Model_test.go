package backupslist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

// cmdDeadline is how long a command gets to produce its message before the
// tests treat it as a timer and move on.
//
// The filter input's cursor blink is the reason. It is a command that sleeps
// for the blink interval and then asks to be run again, so feeding its
// messages back is an infinite loop paced at half a second a lap - which is a
// hang, not a slow test. Nothing this package asserts on is produced by a
// command that has to wait, so abandoning the slow ones costs no coverage.
const cmdDeadline = 50 * time.Millisecond

// emit drains a command into the messages it produces (handles BatchMsg).
func emit(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}

	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()

	select {
	case msg := <-done:
		return collectMsg(msg)
	case <-time.After(cmdDeadline):
		return nil
	}
}

// collectMsg flattens a message, running any batched commands to get at what
// they produce.
//
// tea.BatchMsg is a []tea.Cmd, not a []tea.Msg: its members have to be called.
// This walked them as though they were already messages, which only ever
// worked because tea.Batch hands back a lone command unwrapped - so every test
// asserting on a single-command Update passed, and anything batched was
// invisible.
func collectMsg(msg tea.Msg) []tea.Msg {
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		return []tea.Msg{msg}
	}

	var out []tea.Msg
	for _, child := range batch {
		out = append(out, emit(child)...)
	}

	return out
}

// settle runs the model forward the way the Bubble Tea runtime does: execute
// whatever a message produced, feed the results back, and repeat until nothing
// is left. Filtering needs it - bubbles applies a filter through a command, so
// a test that drops the command types into an input that never narrows
// anything.
func settle(t *testing.T, m Model, msgs ...tea.Msg) Model {
	t.Helper()

	queue := append([]tea.Msg{}, msgs...)

	for range 100 {
		if len(queue) == 0 {
			return m
		}

		msg := queue[0]
		queue = queue[1:]

		updated, cmd := m.Update(msg)
		m = updated.(Model)
		queue = append(queue, emit(cmd)...)
	}

	t.Fatal("the list never settled: a command kept producing messages")

	return m
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

	// Lay the panel out first: an unsized list paginates to one row a page,
	// which is a state the running app never reaches.
	sized, _ := New().Update(cmds.SetBodyLayoutMsg{LeftWidth: 60, RightWidth: 60, Height: 30})
	updated, _ := sized.(Model).Update(cmds.BackupListMsg{Entries: merged})
	m, ok := updated.(Model)
	if !ok {
		t.Fatalf("expected a Model, got %T", updated)
	}
	if len(entriesOf(m)) < 3 {
		t.Fatalf("precondition: %d entries, want at least 3", len(entriesOf(m)))
	}
	if cursor(m) != 0 {
		t.Fatalf("precondition: cursor starts at %d, want 0", cursor(m))
	}

	return m
}

// entriesOf reads the rows back off the list, in order. The panel keeps no
// slice of its own any more - the bubbles list owns the rows - so the tests
// ask it the same way the delegate does.
func entriesOf(m Model) []utils.BackupEntry {
	out := make([]utils.BackupEntry, 0, len(m.list.Items()))
	for _, item := range m.list.Items() {
		if backup, ok := item.(backupItem); ok {
			out = append(out, backup.entry)
		}
	}

	return out
}

// cursor is which row the cursor is on, in the list's own visible space.
func cursor(m Model) int { return m.list.Index() }

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

	entries := entriesOf(m)
	if len(entries) != 3 {
		t.Fatalf("entries: got %d, want 3", len(entries))
	}
	// Newest-first: a strictly older timestamp must not appear before a
	// newer one. Two copies written in the same second share a timestamp
	// prefix and are ordered by name, which is still newest-first.
	for i := 1; i < len(entries); i++ {
		if entries[i-1].Timestamp.Before(entries[i].Timestamp) {
			t.Errorf("entries not newest-first at %d: %q (%s) before %q (%s)",
				i, entries[i-1].Name, entries[i-1].Timestamp, entries[i].Name, entries[i].Timestamp)
		}
	}

	// Each source is represented exactly how many times it was written.
	var composeCount, envCount int
	for _, e := range entries {
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
	if got, want := requested.Source, entriesOf(m)[0].Source; got != want {
		t.Errorf("restore Source = %q, want %q", got, want)
	}
	if got, want := requested.Name, entriesOf(m)[0].Name; got != want {
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

	if !m.ListEmpty() {
		t.Error("the panel does not report itself empty when there are no backups")
	}

	// Nothing is published here, and that is right: the panel had no selection
	// to begin with, so the preview is already on its empty state and a
	// cleared selection would be telling it something it knows. The case that
	// does have to publish - a loaded list reloading to empty - is
	// TestReloadingToAnEmptyStoreClearsTheSelection below.
	if _, published := selectionFrom(cmd); published {
		t.Error("a store that was empty all along published a selection change")
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

			if m = pressKey(t, m, tc.down); cursor(m) != 1 {
				t.Fatalf("after one down the cursor is at %d, want 1", cursor(m))
			}
			if m = pressKey(t, m, tc.down); cursor(m) != 2 {
				t.Fatalf("after two down the cursor is at %d, want 2", cursor(m))
			}
			if m = pressKey(t, m, tc.up); cursor(m) != 1 {
				t.Fatalf("after up the cursor is at %d, want 1", cursor(m))
			}
		})
	}
}

// The cursor stops at both ends rather than wrapping or running off.
func TestTheBackupListStopsAtBothEnds(t *testing.T) {
	m := loadedList(t)

	count := len(entriesOf(m))

	for range count + 3 {
		m = pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if got, want := cursor(m), count-1; got != want {
		t.Errorf("holding down left the cursor at %d, want %d", got, want)
	}

	for range count + 3 {
		m = pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	}
	if cursor(m) != 0 {
		t.Errorf("holding up left the cursor at %d, want 0", cursor(m))
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
	if want := entriesOf(m)[cursor(m)]; sel.Path != want.Path {
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

// A reload that comes back empty clears the preview. The store shrinking to
// nothing is the one way the rows can vanish without an error, and the panel
// holding the bytes is not the panel that knows they are gone.
func TestReloadingToAnEmptyStoreClearsTheSelection(t *testing.T) {
	m := loadedList(t)

	_, cmd := m.Update(cmds.BackupListMsg{Entries: nil})

	sel, published := selectionFrom(cmd)
	if !published {
		t.Fatal("reloading to an empty store published nothing, so the preview keeps the last copy")
	}
	if sel.Name != "" {
		t.Errorf("reloading to an empty store published %q, want the cleared selection", sel.Name)
	}
}

// A reload that lands on the same newest copy publishes nothing. The preview
// is already showing it, and republishing would make it re-read the file off
// disk for no reason - which is what the unconditional publish this replaced
// did on every page switch.
func TestReloadingToTheSameTopCopyPublishesNothing(t *testing.T) {
	m := loadedList(t)
	entries := entriesOf(m)

	_, cmd := m.Update(cmds.BackupListMsg{Entries: entries})

	if sel, published := selectionFrom(cmd); published {
		t.Errorf("an unchanged reload republished %q", sel.Name)
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

// A row is current when its own content hash matches its source's live hash
// from the same read. Several rows can match at once - a restore re-uses
// content that is already in the store - and every one of them is marked.
func TestRowsMatchingTheLiveFileAreMarkedCurrent(t *testing.T) {
	m := New().(Model)

	envSHA := utils.ContentSHA8([]byte("SECRET=1\n"))
	msg := cmds.BackupListMsg{
		Entries: []utils.BackupEntry{
			{Source: "compose", File: "compose.yaml", Name: "20260101T000000.aaaa1111.bak", SHA8: "aaaa1111"},
			{Source: "compose", File: "compose.yaml", Name: "20260102T000000.aaaa1111.bak", SHA8: "aaaa1111"},
			{Source: "compose", File: "compose.yaml", Name: "20260103T000000.bbbb2222.bak", SHA8: "bbbb2222"},
			{Source: ".env", File: ".env", Name: "20260104T000000." + envSHA + ".bak", SHA8: envSHA},
		},
		Live: []utils.LiveSource{
			{Source: "compose", File: "compose.yaml", SHA8: "aaaa1111"},
			{Source: ".env", File: ".env", SHA8: envSHA},
		},
	}

	updated, _ := m.Update(msg)
	m = updated.(Model)

	var marked []int
	for i, item := range m.list.Items() {
		if item.(backupItem).isCurrent {
			marked = append(marked, i)
		}
	}
	want := []int{0, 1, 3}
	if len(marked) != len(want) {
		t.Fatalf("rows marked current = %v, want %v", marked, want)
	}
	for i := range want {
		if marked[i] != want[i] {
			t.Fatalf("rows marked current = %v, want %v", marked, want)
		}
	}
}

// A row is only marked by its own source's live read. A lookup keyed on
// anything less than the row's Source would mark a compose row by a .env
// hash that happens to match, and the two files are different files.
func TestARowIsOnlyMarkedByItsOwnSource(t *testing.T) {
	m := New().(Model)

	envSHA := utils.ContentSHA8([]byte("SECRET=1\n"))
	msg := cmds.BackupListMsg{
		// The compose row carries the .env's hash, and its own source has
		// no live read at all.
		Entries: []utils.BackupEntry{
			{Source: "compose", File: "compose.yaml", Name: "20260101T000000." + envSHA + ".bak", SHA8: envSHA},
		},
		Live: []utils.LiveSource{
			{Source: ".env", File: ".env", SHA8: envSHA},
		},
	}

	updated, _ := m.Update(msg)
	m = updated.(Model)

	for i, item := range m.list.Items() {
		if item.(backupItem).isCurrent {
			t.Errorf("row %d was marked current by another source's hash", i)
		}
	}
}
