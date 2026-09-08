package backuppreviewpanel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/diff"
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

// An .env copy is shown raw, but "raw" is about the bytes, not the colour.
// Text with no SGR of its own lands on the terminal's default foreground,
// which has nothing to do with the active theme - on a light theme that was
// pale grey on a pale panel, and the preview was effectively unreadable.
func TestAnEnvCopyIsRenderedWithAForeground(t *testing.T) {
	m := selectAndLoad(t, New().(Model), entryFor(t, ".env"))

	sized, _ := m.Update(cmds.SetBodyLayoutMsg{LeftWidth: 60, RightWidth: 60, Height: 30})
	frame := sized.(Model).View().Content

	// The bytes are still exactly what a restore would write.
	if !strings.Contains(ansi.Strip(frame), "SECRET=1") {
		t.Fatalf("the .env preview lost its content:\n%s", ansi.Strip(frame))
	}

	want := lipgloss.NewStyle().Foreground(appstyles.Active.TextPrimary).Render("SECRET=1")
	if !strings.Contains(frame, want) {
		t.Error("the .env preview renders its text with no foreground of its own, so it falls back to the terminal's")
	}
}

// The compose path is unchanged: it is syntax highlighted, not flattened to
// one foreground. This is the negative control for the test above - a fix that
// styled everything the same way would pass that one and break this.
func TestAComposeCopyIsStillHighlighted(t *testing.T) {
	m := selectAndLoad(t, New().(Model), entryFor(t, "compose"))

	sized, _ := m.Update(cmds.SetBodyLayoutMsg{LeftWidth: 60, RightWidth: 60, Height: 30})
	frame := sized.(Model).View().Content

	keyed := lipgloss.NewStyle().Foreground(appstyles.Active.Accent).Render("services")
	if !strings.Contains(frame, keyed) {
		t.Error("the compose preview is no longer syntax highlighted")
	}
}

// seedHistory writes the live compose file through two snapshots, so the
// store holds two copies with known contents, and leaves the live file
// itself different from both. The oldest copy is what a "restore to the
// beginning" would put back.
func seedHistory(t *testing.T) (oldest utils.BackupEntry, live string) {
	t.Helper()
	dir := t.TempDir()
	compose := filepath.Join(dir, "compose.yaml")

	for _, contents := range []string{"a\nb\n", "a\nB\n"} {
		if err := os.WriteFile(compose, []byte(contents), 0o644); err != nil {
			t.Fatalf("writing compose: %v", err)
		}
		if err := utils.SnapshotFile(compose); err != nil {
			t.Fatalf("snapshot compose: %v", err)
		}
	}
	// The live file moves on after both copies were taken.
	live = "a\nb\nc\n"
	if err := os.WriteFile(compose, []byte(live), 0o644); err != nil {
		t.Fatalf("writing live compose: %v", err)
	}

	entries, err := utils.ListBackups(compose)
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("ListBackups: %d entries, want 2", len(entries))
	}

	// Both snapshots can land inside the same second, and entries written
	// in the same second then sort by hash, not by write order - so the
	// seeded copy is found by its content hash, never by its position.
	want := utils.ContentSHA8([]byte("a\nb\n"))
	for _, entry := range entries {
		if entry.SHA8 == want {
			return entry, live
		}
	}
	t.Fatalf("seeded copy %s not in the store", want)
	return utils.BackupEntry{}, ""
}

// listMsgFor builds the live half of a list read, the way cmds.GetBackups
// assembles it.
func listMsgFor(entries []utils.BackupEntry, source, contents string) cmds.BackupListMsg {
	return cmds.BackupListMsg{
		Entries: entries,
		Live: []utils.LiveSource{{
			Source:   source,
			File:     "compose.yaml",
			SHA8:     utils.ContentSHA8([]byte(contents)),
			Contents: contents,
		}},
	}
}

// The preview diffs against the live file, in the restore direction:
// Lines(live, copy). A line only the live file has is what restoring the
// copy would remove from it; a line only the copy has is what it would add.
// Everything below that renders the diff depends on this direction being
// right, so it is pinned here where the bytes are known.
func TestThePreviewDiffsAgainstTheLiveFile(t *testing.T) {
	oldest, live := seedHistory(t)

	updated, _ := New().Update(listMsgFor(nil, "compose", live))
	m := updated.(Model)

	m = selectAndLoad(t, m, oldest)

	// live "a\nb\nc\n" against the copy "a\nb\n": restoring would remove c.
	want := []diff.Line{
		{Kind: diff.Equal, Content: "a"},
		{Kind: diff.Equal, Content: "b"},
		{Kind: diff.Delete, Content: "c"},
	}
	if !linesEqual(m.lines, want) {
		t.Errorf("diff = %+v, want %+v", m.lines, want)
	}
}

// A list read that lands without the live file's bytes - a .env that was
// never written, a compose file gone from disk - leaves no diff to compute.
// The panels answer for that; the panel here must not invent a diff out of
// nothing.
func TestNoLiveBytesMeansNoDiff(t *testing.T) {
	m := selectAndLoad(t, New().(Model), entryFor(t, "compose"))

	if m.lines != nil {
		t.Errorf("diff computed with no live bytes on the message: %+v", m.lines)
	}
}

// A live read landing before any copy is selected has nothing to diff
// against either - and must not keep the previous copy's bytes as its other
// side once the selection is cleared.
func TestLiveBytesAloneDoNotDiffAnything(t *testing.T) {
	_, live := seedHistory(t)

	m := selectAndLoad(t, New().(Model), entryFor(t, "compose"))

	updated, _ := m.Update(cmds.SetSelectedBackupMsg(utils.BackupEntry{}))
	m = updated.(Model)
	updated, _ = m.Update(listMsgFor(nil, "compose", live))
	m = updated.(Model)

	if m.lines != nil {
		t.Errorf("a live read with nothing selected produced a diff: %+v", m.lines)
	}
}

// A re-list after a write replaces the live side without moving the cursor,
// so no new selection publish and no new .bak read fire. The diff must
// follow the new live bytes anyway, or the panel keeps answering a question
// about a file that no longer exists.
func TestARelistRecomputesTheDiffAgainstTheNewLiveFile(t *testing.T) {
	oldest, live := seedHistory(t)

	updated, _ := New().Update(listMsgFor([]utils.BackupEntry{oldest}, "compose", live))
	m := updated.(Model)
	m = selectAndLoad(t, m, oldest)
	if len(m.lines) == 0 {
		t.Fatal("precondition: no diff after the first load")
	}

	newLive := "x\n"
	updated, _ = m.Update(listMsgFor([]utils.BackupEntry{oldest}, "compose", newLive))
	m = updated.(Model)

	// live "x\n" against the copy "a\nb\n": restoring would empty it first.
	want := []diff.Line{
		{Kind: diff.Delete, Content: "x"},
		{Kind: diff.Insert, Content: "a"},
		{Kind: diff.Insert, Content: "b"},
	}
	if !linesEqual(m.lines, want) {
		t.Errorf("diff after re-list = %+v, want %+v", m.lines, want)
	}
}

// linesEqual compares field by field: diff.Line carries a slice, so it
// cannot be compared with !=.
func linesEqual(got, want []diff.Line) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i].Kind != want[i].Kind || got[i].Content != want[i].Content {
			return false
		}
	}
	return true
}
