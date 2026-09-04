package backupslist

import (
	"fmt"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/utils"
)

// syntheticEntries builds n rows without touching the disk. The scroll
// behaviour depends only on how many entries there are, not on what is in
// them, and the store's ceiling is 500 per source - far too many to seed by
// snapshotting real files.
func syntheticEntries(n int) []utils.BackupEntry {
	entries := make([]utils.BackupEntry, n)
	base := time.Date(2026, 8, 11, 9, 15, 0, 0, time.UTC)

	for i := range entries {
		entries[i] = utils.BackupEntry{
			Source:    "compose",
			Name:      fmt.Sprintf("%s.%08x.bak", base.Add(-time.Duration(i)*time.Minute).Format("20060102T150405"), i),
			Timestamp: base.Add(-time.Duration(i) * time.Minute),
			SHA8:      fmt.Sprintf("%08x", i),
			Path:      fmt.Sprintf("/nowhere/%d.bak", i),
		}
	}

	return entries
}

// listWithRows returns a sized list holding n entries, plus how many rows fit
// on screen. The capacity is read back off the model rather than hardcoded,
// so a change to the panel's padding does not silently invalidate the
// boundary cases below.
func listWithRows(t *testing.T, n int) (Model, int) {
	t.Helper()

	m := New().(Model)
	sized, _ := m.Update(cmds.SetBodyLayoutMsg{LeftWidth: 60, RightWidth: 60, Height: 24})
	m = sized.(Model)

	loaded, _ := m.Update(cmds.BackupListMsg{Entries: syntheticEntries(n)})
	m = loaded.(Model)

	visibleRows := m.visibleRows()
	if visibleRows < 2 {
		t.Fatalf("test panel only fits %d rows; too small to exercise scrolling", visibleRows)
	}

	return m, visibleRows
}

// assertCursorVisible is the invariant the whole phase exists for: whatever
// the cursor is on, both of that row's lines are inside the viewport's
// window. Before the viewport, this held only for the first screenful.
func assertCursorVisible(t *testing.T, m Model, whenDoing string) {
	t.Helper()

	first := m.rowOffset
	last := m.rowOffset + m.visibleRows() - 1

	if m.selectedIdx < first || m.selectedIdx > last {
		t.Errorf("after %s the cursor is on row %d but the window shows rows %d-%d",
			whenDoing, m.selectedIdx, first, last)
	}
}

// A history shorter than the panel never scrolls: there is nothing below the
// fold to scroll to.
func TestAShortListNeverScrolls(t *testing.T) {
	m, visibleRows := listWithRows(t, 2)

	for range visibleRows + 5 {
		m = pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
		if got := m.rowOffset; got != 0 {
			t.Fatalf("a list of 2 rows scrolled to row %d", got)
		}
	}
}

// Walking off the bottom of the window scrolls by exactly one row, so the
// list moves as little as it can to keep up with the cursor.
func TestWalkingPastTheFoldScrollsOneRow(t *testing.T) {
	m, visibleRows := listWithRows(t, 40)

	// Walk to the last row that is already on screen. Nothing should move.
	for range visibleRows - 1 {
		m = pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if got := m.rowOffset; got != 0 {
		t.Fatalf("the cursor reached the fold and the list had already scrolled to row %d", got)
	}
	assertCursorVisible(t, m, "walking to the fold")

	// One more row is one row past the fold.
	m = pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	if got, want := m.rowOffset, 1; got != want {
		t.Errorf("stepping past the fold scrolled to row %d, want %d", got, want)
	}
	assertCursorVisible(t, m, "stepping one past the fold")
}

// The cursor stays visible the whole way down a long history and the whole
// way back up, not just at the ends.
func TestTheCursorStaysVisibleThroughoutALongHistory(t *testing.T) {
	m, _ := listWithRows(t, 60)

	for i := range 59 {
		m = pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
		assertCursorVisible(t, m, fmt.Sprintf("%d presses of down", i+1))
	}

	for i := range 59 {
		m = pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
		assertCursorVisible(t, m, fmt.Sprintf("%d presses of up", i+1))
	}
}

// G jumps to the oldest copy and scrolls it into view. This is the case that
// was most clearly broken before: the last row of a long history was
// guaranteed to be off-screen, so G moved the cursor somewhere invisible and
// loaded a preview for a row the user could not see.
func TestEndJumpsToTheLastRowAndScrollsToIt(t *testing.T) {
	m, visibleRows := listWithRows(t, 40)

	m = pressKey(t, m, tea.KeyPressMsg{Code: 'G', Text: "G"})

	if got, want := m.selectedIdx, 39; got != want {
		t.Fatalf("G left the cursor at %d, want %d", got, want)
	}
	assertCursorVisible(t, m, "pressing G")

	// The last row sits at the bottom of the window, with no blank rows
	// scrolled past it.
	if got, want := m.rowOffset, 40-visibleRows; got != want {
		t.Errorf("G scrolled to row %d, want %d", got, want)
	}
}

// g comes back to the newest copy and scrolls the list back to the top.
func TestHomeReturnsToTheTop(t *testing.T) {
	m, _ := listWithRows(t, 40)

	m = pressKey(t, m, tea.KeyPressMsg{Code: 'G', Text: "G"})
	if m.rowOffset == 0 {
		t.Fatal("precondition: the list never scrolled, so returning to the top proves nothing")
	}

	m = pressKey(t, m, tea.KeyPressMsg{Code: 'g', Text: "g"})

	if m.selectedIdx != 0 {
		t.Errorf("g left the cursor at %d, want 0", m.selectedIdx)
	}
	if got := m.rowOffset; got != 0 {
		t.Errorf("g left the list scrolled to row %d, want the top", got)
	}
}

// A row exactly at the fold is fully visible, both of its lines. A row is two
// lines tall, so an off-by-one here shows the timestamp and clips the sha
// under it.
func TestARowAtTheFoldIsFullyVisible(t *testing.T) {
	m, visibleRows := listWithRows(t, 40)

	for range visibleRows - 1 {
		m = pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	}

	// The row at the fold is rendered whole - both its lines - because the
	// window counts whole rows and a partial row is never counted in.
	if got, want := m.selectedIdx, m.rowOffset+visibleRows-1; got != want {
		t.Fatalf("the cursor is on row %d, want the last row of the window (%d)", got, want)
	}

	usable := chrome.PanelBodyHeight(m.panelHeight) - headerHeight
	if rendered := visibleRows * rowHeight; rendered > usable {
		t.Errorf("the window renders %d lines into %d, so the last row is clipped", rendered, usable)
	}
}

// Shrinking the panel pulls the cursor back into view rather than leaving it
// below a fold that just moved up.
func TestShrinkingThePanelKeepsTheCursorVisible(t *testing.T) {
	m, visibleRows := listWithRows(t, 40)

	for range visibleRows - 1 {
		m = pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	}

	resized, _ := m.Update(cmds.SetBodyLayoutMsg{LeftWidth: 60, RightWidth: 60, Height: 14})
	m = resized.(Model)

	assertCursorVisible(t, m, "shrinking the panel")
}

// Reloading the store puts the cursor back on the newest copy and scrolls
// there with it, rather than leaving the window where the old cursor was.
func TestReloadingScrollsBackToTheTop(t *testing.T) {
	m, _ := listWithRows(t, 40)

	m = pressKey(t, m, tea.KeyPressMsg{Code: 'G', Text: "G"})
	if m.rowOffset == 0 {
		t.Fatal("precondition: the list never scrolled")
	}

	reloaded, _ := m.Update(cmds.BackupListMsg{Entries: syntheticEntries(40)})
	m = reloaded.(Model)

	if m.selectedIdx != 0 {
		t.Errorf("a reload left the cursor at %d, want 0", m.selectedIdx)
	}
	if got := m.rowOffset; got != 0 {
		t.Errorf("a reload left the list scrolled to row %d, want the top", got)
	}
}

// BenchmarkRenderRowsAtStoreCeiling measures a frame at the store's ceiling:
// MaxBackupsPerSource for compose and the same again for .env.
//
// The number that matters is that it does not grow with the history. An
// earlier draft rendered every row into a viewport and scrolled that, which
// re-rendered all 1000 rows on every cursor move and measured 39ms per
// keystroke - past a frame, and visibly laggy on a held key. Rendering only
// the window costs the same whether the store holds ten copies or a thousand.
func BenchmarkRenderRowsAtStoreCeiling(b *testing.B) {
	m := New().(Model)
	sized, _ := m.Update(cmds.SetBodyLayoutMsg{LeftWidth: 60, RightWidth: 60, Height: 40})
	m = sized.(Model)

	loaded, _ := m.Update(cmds.BackupListMsg{Entries: syntheticEntries(2 * utils.MaxBackupsPerSource)})
	m = loaded.(Model)

	for i := 0; b.Loop(); i++ {
		m.selectedIdx = i % len(m.entries)
		m.ensureCursorVisible()
		_ = m.View()
	}
}
