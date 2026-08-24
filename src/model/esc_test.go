package model

import (
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/cmds"
)

// escKey is the escape key as a terminal delivers it.
func escKey() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeyEscape}
}

// withFilterApplied drives the focused groups list past typing to an applied
// filter: /, a letter, enter.
func withFilterApplied(t *testing.T, m AppModel) AppModel {
	t.Helper()

	m = filtering(t, m)
	m = drive(m, letter('m'), tea.KeyPressMsg{Code: tea.KeyEnter})

	if m.keyboardOwned() {
		t.Fatal("the list still owns the keyboard after the filter was applied")
	}

	return m
}

// clearedFilter reports whether msgs carry the list's announcement that its
// filter is gone.
func clearedFilter(msgs []tea.Msg) bool {
	for _, msg := range msgs {
		if state, ok := msg.(cmds.SetListFilterStateMsg); ok && list.FilterState(state) == list.Unfiltered {
			return true
		}
	}

	return false
}

// A list holding an applied filter keeps esc - it is the only way back to the
// full rows - so esc clears the filter rather than touching the selection.
func TestEscClearsAnAppliedFilter(t *testing.T) {
	m := withFilterApplied(t, homeWithGroups(t))

	updated, cmd := m.Update(escKey())
	m = updated.(AppModel)

	if !clearedFilter(collect(cmd)) {
		t.Error("esc did not clear the applied filter")
	}
	// No focus to move, and the selection is untouched.
	if m.selection.groupName != "" {
		t.Error("esc cleared the filter but also changed the selection")
	}
}

// With a group selected and no banner or filter in the way, esc deselects the
// current group - the "back" rung of the ladder now that there is no panel to
// return focus to.
func TestEscOnASelectedGroupDeselectsIt(t *testing.T) {
	m := homeWithGroups(t)
	m.selection.groupName = "core"

	updated, cmd := m.Update(escKey())
	m = updated.(AppModel)

	if m.selection.groupName != "" {
		t.Errorf("esc did not deselect the group: selection = %q", m.selection.groupName)
	}
	for _, msg := range collect(cmd) {
		if _, ok := msg.(cmds.SetSelectedGroupMsg); ok {
			return
		}
	}
	t.Error("esc did not broadcast the deselection")
}

// On an unfiltered list with no selection, esc is nobody's key: it produces no
// filter state change and no deselection.
func TestEscOnAnUnfilteredListDoesNothing(t *testing.T) {
	m := homeWithGroups(t)
	m.selection.groupName = ""

	_, cmd := m.Update(escKey())

	for _, msg := range collect(cmd) {
		switch msg.(type) {
		case cmds.SetListFilterStateMsg:
			t.Errorf("esc on an unfiltered list produced %T", msg)
		}
	}
	if m.selection.groupName != "" {
		t.Error("esc changed the selection on an unfiltered list")
	}
}
