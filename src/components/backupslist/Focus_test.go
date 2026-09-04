package backupslist

import (
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/cmds"
)

// focus tells the list which half of the page the arrows are driving, the way
// AppModel's broadcast does.
func focus(t *testing.T, m Model, to apptypes.BackupsFocus) Model {
	t.Helper()

	updated, _ := m.Update(cmds.SetBackupsFocusMsg(to))
	next, ok := updated.(Model)
	if !ok {
		t.Fatalf("expected a Model, got %T", updated)
	}

	return next
}

// The list starts focused. The page opens on it, and a list that had to be
// tabbed into before the arrows worked would be a page whose first keystroke
// does nothing.
func TestListStartsFocused(t *testing.T) {
	if !New().(Model).focused {
		t.Error("the version list should hold focus when the page opens")
	}
}

// Unfocused, the arrows belong to the preview. Both panels receive every
// keystroke, so the list has to decline the ones it does not own or a single
// arrow press moves two things at once.
//
// This is the whole reason the list is not simply handed every message: the
// bubbles list would answer all of these.
func TestUnfocusedListIgnoresTheListKeys(t *testing.T) {
	m := focus(t, listWith(t, 20), apptypes.BackupsPreview)

	for _, msg := range []tea.KeyPressMsg{
		{Code: tea.KeyDown},
		{Code: tea.KeyUp},
		{Code: 'j', Text: "j"},
		{Code: 'k', Text: "k"},
		{Code: 'g', Text: "g"},
		{Code: 'G', Text: "G"},
		{Code: tea.KeyPgDown},
		{Code: tea.KeyPgUp},
		{Code: 'l', Text: "l"},
		{Code: 'h', Text: "h"},
		{Code: '/', Text: "/"},
	} {
		moved, cmd := m.Update(msg)

		if got := cursor(moved.(Model)); got != cursor(m) {
			t.Errorf("%v moved the cursor from %d to %d while the preview held focus",
				msg, cursor(m), got)
		}
		if moved.(Model).FilterState() != list.Unfiltered {
			t.Errorf("%v started a filter while the preview held focus", msg)
		}
		if cmd != nil {
			t.Errorf("%v published something while the preview held focus", msg)
		}
	}

	// Negative control: the same keys work once focus comes back, so the
	// assertions above are about focus rather than about a list that has
	// quietly stopped answering keys at all.
	back := pressKey(t, focus(t, m, apptypes.BackupsList), tea.KeyPressMsg{Code: tea.KeyDown})
	if cursor(back) != 1 {
		t.Errorf("↓ should move the cursor to row 1 with the list focused, got %d", cursor(back))
	}
}

// Non-key messages reach the list whichever panel is lit. Withholding them
// would leave an unfocused list sized for the previous terminal, or holding
// rows the store no longer has.
func TestAnUnfocusedListStillResizesAndReloads(t *testing.T) {
	m := focus(t, listWith(t, 4), apptypes.BackupsPreview)

	resized, _ := m.Update(cmds.SetBodyLayoutMsg{LeftWidth: 30, RightWidth: 90, Height: 40})
	m = resized.(Model)
	if m.panelWidth != 30 {
		t.Errorf("an unfocused list ignored a resize: width %d, want 30", m.panelWidth)
	}

	reloaded, _ := m.Update(cmds.BackupListMsg{Entries: syntheticEntries(9)})
	if got := len(entriesOf(reloaded.(Model))); got != 9 {
		t.Errorf("an unfocused list ignored a reload: %d rows, want 9", got)
	}
}

// Restore is not gated on focus. Focus decides what the arrows drive; it does
// not move the selection, and the selection is what a restore acts on - so r
// works from either half of the page.
func TestRestoreWorksWithThePreviewFocused(t *testing.T) {
	m := focus(t, listWith(t, 5), apptypes.BackupsPreview)

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	if cmd == nil {
		t.Fatal("r issued no restore while the preview held focus")
	}

	if _, ok := selectionFrom(cmd); ok {
		t.Error("r published a selection; it should only request the restore")
	}

	var requested bool
	for _, msg := range emit(cmd) {
		if _, ok := msg.(cmds.RequestRestoreBackupMsg); ok {
			requested = true
		}
	}
	if !requested {
		t.Error("r produced no restore request while the preview held focus")
	}
}

// enter was dropped from the restore binding: it is too easy to hit by reflex
// while navigating, for an action that overwrites a live file.
func TestEnterNoLongerRestores(t *testing.T) {
	m := listWith(t, 5)

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	for _, msg := range emit(cmd) {
		if _, ok := msg.(cmds.RequestRestoreBackupMsg); ok {
			t.Error("enter still requests a restore")
		}
	}
}

// While a filter is being typed the list owns the whole keyboard, so r is a
// letter rather than a verb. Same rule as the services list's d.
func TestRestoreDoesNotFireWhileFiltering(t *testing.T) {
	m := pressKey(t, listWith(t, 5), tea.KeyPressMsg{Code: '/', Text: "/"})

	if !m.OwnsKeyboard() {
		t.Fatal("precondition: / did not start a filter")
	}

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	for _, msg := range emit(cmd) {
		if _, ok := msg.(cmds.RequestRestoreBackupMsg); ok {
			t.Error("r requested a restore while it was being typed into the filter")
		}
	}
}
