package backupslist

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/utils"
)

// syntheticEntries builds n rows without touching the disk. Row rendering
// depends only on how many entries there are, not on what is in them, and the
// store's ceiling is 500 per source - far too many to seed by snapshotting
// real files.
func syntheticEntries(n int) []utils.BackupEntry {
	entries := make([]utils.BackupEntry, n)
	base := time.Date(2026, 8, 11, 9, 15, 0, 0, time.UTC)

	for i := range entries {
		entries[i] = utils.BackupEntry{
			Source:    "compose",
			File:      "compose.yaml",
			Name:      fmt.Sprintf("%s.%08x.bak", base.Add(-time.Duration(i)*time.Minute).Format("20060102T150405"), i),
			Timestamp: base.Add(-time.Duration(i) * time.Minute),
			SHA8:      fmt.Sprintf("%08x", i),
			Path:      fmt.Sprintf("/nowhere/%d.bak", i),
		}
	}

	return entries
}

// listWith returns a sized panel holding n entries.
func listWith(t *testing.T, n int) Model {
	t.Helper()

	sized, _ := New().Update(cmds.SetBodyLayoutMsg{LeftWidth: 60, RightWidth: 60, Height: 24})
	loaded, _ := sized.(Model).Update(cmds.BackupListMsg{Entries: syntheticEntries(n)})

	return loaded.(Model)
}

// A row names the live file it would restore over, and does not carry the
// sha. Two compose files can sit in one directory with separate histories,
// so a row that said only "compose" could not tell them apart; the sha
// identifies content, which is the preview panel's job beside this list.
func TestARowNamesTheFileAndNotTheSha(t *testing.T) {
	entries := syntheticEntries(1)
	entries[0].File = "compose.yml"

	sized, _ := New().Update(cmds.SetBodyLayoutMsg{LeftWidth: 60, RightWidth: 60, Height: 24})
	loaded, _ := sized.(Model).Update(cmds.BackupListMsg{Entries: entries})

	frame := ansi.Strip(loaded.(Model).View().Content)

	if !strings.Contains(frame, "compose.yml") {
		t.Errorf("the row does not name the live file:\n%s", frame)
	}
	if strings.Contains(frame, entries[0].SHA8) {
		t.Errorf("the row still carries the sha %q, which belongs to the preview:\n%s", entries[0].SHA8, frame)
	}
	// The timestamp says which zone it is in, since no column header does.
	if !strings.Contains(frame, "UTC") {
		t.Errorf("the row's timestamp does not say it is UTC:\n%s", frame)
	}
}

// The panel reports how much history it holds. It was the panel title's
// right-hand accessory before the bubbles list took the job; the list's own
// status bar says it now, which is why the status bar is on here and off on
// the groups and services lists.
func TestThePanelReportsTheVersionCount(t *testing.T) {
	frame := ansi.Strip(listWith(t, 7).View().Content)

	if !strings.Contains(frame, "7 versions") {
		t.Errorf("the panel does not report its version count:\n%s", frame)
	}

	// Negative control: the count is the number of rows, not a constant.
	if one := ansi.Strip(listWith(t, 1).View().Content); !strings.Contains(one, "1 version") {
		t.Errorf("a single copy is not reported as \"1 version\":\n%s", one)
	}
}

// BenchmarkRenderRowsAtStoreCeiling renders a full frame with the store at its
// ceiling - MaxBackupsPerSource for compose and the same again for .env.
//
// The number that matters is that it does not grow with the history. An
// earlier draft rendered every row into a viewport and scrolled that, which
// re-rendered all 1000 rows on every cursor move and measured 39ms per
// keystroke - past a frame, and visibly laggy on a held key. The bubbles list
// renders one page, so the cost is the same whether the store holds ten copies
// or a thousand.
func BenchmarkRenderRowsAtStoreCeiling(b *testing.B) {
	sized, _ := New().Update(cmds.SetBodyLayoutMsg{LeftWidth: 60, RightWidth: 60, Height: 40})
	loaded, _ := sized.(Model).Update(cmds.BackupListMsg{Entries: syntheticEntries(2 * utils.MaxBackupsPerSource)})
	m := loaded.(Model)

	for b.Loop() {
		next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		m = next.(Model)
		_ = m.View()
	}
}

// Pagination has to settle in one go: loading rows and then doing anything
// that re-runs the list's own pagination must not change how many rows a page
// holds.
//
// list.updatePagination sizes a page against a height it has taken the
// paginator's row out of - but list.paginationView renders nothing, and so
// measures a bare line, until it knows there is more than one page. The first
// pass after rows arrive therefore fits one row too many, and the next pass
// silently disagrees with it: the panel drew four rows a page while focused
// and three the moment anything touched the delegate. Model.setItems runs the
// pass twice so the answer is settled before anything renders.
func TestPaginationIsStableAfterLoading(t *testing.T) {
	m := listWith(t, 40)

	perPage := m.list.Paginator.PerPage
	if m.list.Paginator.TotalPages < 2 {
		t.Fatalf("precondition: 40 rows fit on one page (%d per page); nothing to paginate", perPage)
	}

	// Any of these re-runs the list's pagination. None of them changes what
	// the panel is showing, so none of them may change how it is paged.
	for _, repaginate := range []struct {
		name string
		do   func(Model) Model
	}{
		{"a focus change", func(m Model) Model { return focus(t, m, apptypes.BackupsPreview) }},
		{"a resize to the same size", func(m Model) Model {
			next, _ := m.Update(cmds.SetBodyLayoutMsg{LeftWidth: 60, RightWidth: 60, Height: 24})
			return next.(Model)
		}},
	} {
		if got := repaginate.do(m).list.Paginator.PerPage; got != perPage {
			t.Errorf("%s changed the page from %d rows to %d; pagination had not settled",
				repaginate.name, perPage, got)
		}
	}
}

// listMarkedCurrent returns a sized panel whose row at markIndex matches the
// live file's hash for its source, so the delegate draws it as current.
func listMarkedCurrent(t *testing.T, n, markIndex int, leftWidth int) Model {
	t.Helper()

	entries := syntheticEntries(n)
	sized, _ := New().Update(cmds.SetBodyLayoutMsg{LeftWidth: leftWidth, RightWidth: 60, Height: 24})
	loaded, _ := sized.(Model).Update(cmds.BackupListMsg{
		Entries: entries,
		Live: []utils.LiveSource{{
			Source: "compose",
			File:   "compose.yaml",
			SHA8:   entries[markIndex].SHA8,
		}},
	})

	return loaded.(Model)
}

// The copy that matches the live file says so, on the title line, exactly
// once: the page's one verb is restore, and the first question about a copy
// is whether restoring it would change anything at all.
func TestTheCurrentRowSaysSo(t *testing.T) {
	m := listMarkedCurrent(t, 3, 0, 60)

	frame := m.View().Content
	plain := ansi.Strip(frame)

	if !strings.Contains(plain, "compose.yaml (current)") {
		t.Errorf("the copy matching the live file is not marked:\n%s", plain)
	}
	if n := strings.Count(plain, "(current)"); n != 1 {
		t.Errorf("%d rows claim to be current, want 1:\n%s", n, plain)
	}

	// The marker is not part of the cursor row's bold run - the title
	// carries the bold, the marker is a quiet suffix beside it.
	if strings.Contains(frame, "\x1b[1m (current") || strings.Contains(frame, "\x1b[1m(current") {
		t.Error("the (current) marker is inside the title's bold run")
	}
}

// A marked row that is not under the cursor runs the marker one emphasis
// step below the title, on the row's own background - never the accent,
// which is the cursor bar's channel alone.
func TestAnUnfocusedRowRunsTheMarkerDimmer(t *testing.T) {
	m := listMarkedCurrent(t, 3, 2, 60)

	rowBg := chrome.ListRowBgOn(false, chrome.PanelBgFor(true))
	want := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextMuted).
		Background(rowBg).
		Render(" (current)")
	if !strings.Contains(m.View().Content, want) {
		t.Error("a marked row off the cursor does not run its marker one step below the title")
	}
}

// On a panel too narrow for the filename and the marker, the marker is the
// first thing to go, and it goes whole: the title is the row's identity, and
// a partial "(curr…" is noise.
func TestTheMarkerIsDroppedWholeBeforeTheTitle(t *testing.T) {
	m := listMarkedCurrent(t, 1, 0, 20)

	plain := ansi.Strip(m.View().Content)

	if !strings.Contains(plain, "compose.yaml") {
		t.Errorf("the title did not survive the narrow panel:\n%s", plain)
	}
	if strings.Contains(plain, "(current") {
		t.Errorf("a marker the panel cannot fit whole was drawn anyway:\n%s", plain)
	}
}
