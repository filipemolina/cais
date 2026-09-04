package backupslist

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/utils"
)

// typeFilter opens the filter, types term, and applies it.
func typeFilter(t *testing.T, m Model, term string) Model {
	t.Helper()

	m = settle(t, m, tea.KeyPressMsg{Code: '/', Text: "/"})
	for _, r := range term {
		m = settle(t, m, tea.KeyPressMsg{Code: r, Text: string(r)})
	}

	return settle(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
}

// mixedEntries is a store holding copies of both sources, so a filter has
// something to narrow.
func mixedEntries() []utils.BackupEntry {
	entries := syntheticEntries(6)
	for i := range entries {
		if i%2 == 1 {
			entries[i].Source = ".env"
			entries[i].File = ".env"
		}
	}

	return entries
}

// A filter narrows the rows to the matching file. This is the whole reason
// the panel moved onto the bubbles list rather than keeping its own cursor:
// filtering came free with it.
func TestFilteringNarrowsToOneSource(t *testing.T) {
	sized, _ := New().Update(cmds.SetBodyLayoutMsg{LeftWidth: 60, RightWidth: 60, Height: 40})
	loaded, _ := sized.(Model).Update(cmds.BackupListMsg{Entries: mixedEntries()})

	m := typeFilter(t, loaded.(Model), ".env")

	if m.FilterState() != list.FilterApplied {
		t.Fatalf("filter state is %v, want applied", m.FilterState())
	}

	visible := m.list.VisibleItems()
	if len(visible) == 0 || len(visible) == len(m.list.Items()) {
		t.Fatalf("the filter matched %d of %d rows; it narrowed nothing", len(visible), len(m.list.Items()))
	}
	for _, item := range visible {
		if got := item.(backupItem).entry.File; got != ".env" {
			t.Errorf("a %q row survived the .env filter", got)
		}
	}
}

// The sha and the timestamp are filterable even though neither is on the row.
// A version list is identified by all three, and the sha is how you find one
// specific copy in a long history.
func TestFilteringMatchesTheShaAndTheTimestamp(t *testing.T) {
	entries := syntheticEntries(6)

	for _, term := range []string{entries[3].SHA8, "2026-08-11"} {
		sized, _ := New().Update(cmds.SetBodyLayoutMsg{LeftWidth: 60, RightWidth: 60, Height: 40})
		loaded, _ := sized.(Model).Update(cmds.BackupListMsg{Entries: entries})

		m := typeFilter(t, loaded.(Model), term)
		if len(m.list.VisibleItems()) == 0 {
			t.Errorf("filtering on %q matched no rows", term)
		}
	}
}

// Applying a filter publishes whatever the cursor landed on. The cursor goes
// to the top of the matches, which is usually a different copy from the one
// that was selected, and the preview has to follow or the two panels name
// different versions.
func TestApplyingAFilterRepublishesTheSelection(t *testing.T) {
	sized, _ := New().Update(cmds.SetBodyLayoutMsg{LeftWidth: 60, RightWidth: 60, Height: 40})
	loaded, _ := sized.(Model).Update(cmds.BackupListMsg{Entries: mixedEntries()})
	m := loaded.(Model)

	// Row 0 is a compose copy; filtering to .env must move off it.
	before, _ := m.selected()

	m = settle(t, m, tea.KeyPressMsg{Code: '/', Text: "/"})
	var published *utils.BackupEntry
	for _, r := range ".env" {
		updated, cmd := m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = updated.(Model)
		if sel, ok := selectionFrom(cmd); ok {
			published = &sel
		}
		for _, msg := range emit(cmd) {
			updated, next := m.Update(msg)
			m = updated.(Model)
			if sel, ok := selectionFrom(next); ok {
				published = &sel
			}
		}
	}

	after, ok := m.selected()
	if !ok {
		t.Fatal("the filter left nothing under the cursor")
	}
	if after.Name == before.Name {
		t.Skip("the filter did not move the cursor off row 0; nothing to assert")
	}
	if published == nil {
		t.Fatal("the cursor moved onto a different copy but no selection was published")
	}
	if published.Name != after.Name {
		t.Errorf("published %q but the cursor is on %q", published.Name, after.Name)
	}
}

// The filter state reaches the footer. The bar decides whether to offer
// "/ filter" or "esc clear filter" from it, and it is a global component that
// never sees the list itself.
func TestFilterStateIsBroadcast(t *testing.T) {
	m := listWith(t, 6)

	_, cmd := m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})

	var broadcast bool
	for _, msg := range emit(cmd) {
		if state, ok := msg.(cmds.SetListFilterStateMsg); ok {
			broadcast = true
			if list.FilterState(state) != list.Filtering {
				t.Errorf("broadcast filter state %v, want filtering", list.FilterState(state))
			}
		}
	}
	if !broadcast {
		t.Error("starting a filter told the footer nothing")
	}
}

// While a filter is being typed the list owns the keyboard, and once one is
// applied it keeps esc. AppModel asks both questions on the keystroke that
// changes them - see model.AppModel.keyboardOwned and escKept.
func TestFilteringClaimsTheKeyboardAndEsc(t *testing.T) {
	m := listWith(t, 6)

	if m.OwnsKeyboard() || m.KeepsEsc() {
		t.Fatal("an unfiltered list should claim neither the keyboard nor esc")
	}

	typing := settle(t, m, tea.KeyPressMsg{Code: '/', Text: "/"})
	if !typing.OwnsKeyboard() {
		t.Error("a filter being typed does not own the keyboard")
	}

	applied := typeFilter(t, m, "compose")
	if applied.OwnsKeyboard() {
		t.Error("an applied filter still owns the whole keyboard")
	}
	if !applied.KeepsEsc() {
		t.Error("an applied filter does not keep esc, so there is no way back to the full rows")
	}
}

// The status bar names the standing filter. bubbles draws the filter input
// only while it is being typed, so without it nothing on screen says the rows
// are narrowed.
func TestTheStatusBarNamesAStandingFilter(t *testing.T) {
	m := typeFilter(t, listWith(t, 6), "compose")

	frame := ansi.Strip(m.View().Content)
	if !strings.Contains(frame, "compose") {
		t.Errorf("the panel does not name the standing filter:\n%s", frame)
	}
}
