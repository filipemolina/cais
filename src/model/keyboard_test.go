package model

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components"
)

// letter is a plain keystroke as a terminal delivers it.
func letter(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

// homeWithGroups is the app on Home, laid out, with a groups list long enough
// to be worth filtering and the left panel focused.
func homeWithGroups(t *testing.T) AppModel {
	t.Helper()

	m := applyLayout(startup(120, 40))
	m = drive(m, cmds.SetGroupsListMsg{"core", "media", "monitoring"})

	if m.keyboardOwned() {
		t.Fatal("nothing should own the keyboard before a filter is started")
	}

	return m
}

// filtering starts a filter on the focused groups list, the way pressing / does.
func filtering(t *testing.T, m AppModel) AppModel {
	t.Helper()

	m = drive(m, letter('/'))
	if !m.keyboardOwned() {
		t.Fatal("/ did not hand the keyboard to the list")
	}

	return m
}

// While a filter is being typed, q is a letter. Quitting on it was the worst of
// the default-keymap collisions: the app exited from under a user who was trying
// to search.
func TestQuitYieldsToAFilteringList(t *testing.T) {
	m := filtering(t, homeWithGroups(t))

	_, cmd := m.Update(letter('q'))

	for _, msg := range collect(cmd) {
		if _, ok := msg.(tea.QuitMsg); ok {
			t.Fatal("q quit the app while the list was being filtered")
		}
	}
}

// Same for the page chords: alt+s while typing a filter would swap the page out
// from under the search.
func TestPageChordsYieldToAFilteringList(t *testing.T) {
	m := filtering(t, homeWithGroups(t))

	_, cmd := m.Update(altKey([]rune(apptypes.PageShortcut("Services"))[0]))

	if got := activePageFrom(collect(cmd)); got != "" {
		t.Errorf("a page chord switched to %q while the list was being filtered", got)
	}
}

// ctrl+c is the exception that makes the rule safe: it quits whatever owns the
// keyboard, so no state can trap the user.
func TestForceQuitBeatsEveryClaimOnTheKeyboard(t *testing.T) {
	forceQuit := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}

	quits := func(t *testing.T, m AppModel) bool {
		t.Helper()

		_, cmd := m.Update(forceQuit)
		for _, msg := range collect(cmd) {
			if _, ok := msg.(tea.QuitMsg); ok {
				return true
			}
		}

		return false
	}

	t.Run("while a list is filtering", func(t *testing.T) {
		if !quits(t, filtering(t, homeWithGroups(t))) {
			t.Error("ctrl+c did not quit while the list was being filtered")
		}
	})

	t.Run("while a modal is open", func(t *testing.T) {
		m := homeWithGroups(t)
		m.activeModal = components.ConfirmModal("Delete group \"core\"? (y/n)", nil)

		if !quits(t, m) {
			t.Error("ctrl+c did not quit while a modal was open")
		}
	})

	t.Run("with nothing in the way", func(t *testing.T) {
		if !quits(t, homeWithGroups(t)) {
			t.Error("ctrl+c did not quit the idle app")
		}
	})
}

// Once the filter is applied the list is back to being a list: the cursor is in
// the rows, so the panel keys and q mean what they always mean.
func TestApplyingTheFilterHandsTheKeyboardBack(t *testing.T) {
	m := filtering(t, homeWithGroups(t))

	m = drive(m, letter('m'), tea.KeyPressMsg{Code: tea.KeyEnter})

	if m.keyboardOwned() {
		t.Error("the list still owns the keyboard after the filter was applied")
	}
}
