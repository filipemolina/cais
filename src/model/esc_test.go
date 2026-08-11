package model

import (
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/constants"
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

// The first constraint from the list keymap work: a focused list holding an
// applied filter keeps esc - it is the only way back to the full rows - and
// global "back" has to yield rather than strand the filter on an unfocused
// panel.
func TestEscClearsAnAppliedFilterBeforeItMovesFocus(t *testing.T) {
	m := withFilterApplied(t, homeWithGroups(t))

	if got := m.focusedComponent; got != constants.COMPONENT_BODY_LIST {
		t.Fatalf("precondition: expected the list focused, got %d", got)
	}

	updated, cmd := m.Update(escKey())
	m = updated.(AppModel)

	if got := m.focusedComponent; got != constants.COMPONENT_BODY_LIST {
		t.Errorf("esc moved focus to component %d instead of clearing the filter", got)
	}
	if !clearedFilter(collect(cmd)) {
		t.Error("esc did not clear the applied filter")
	}
}

// What is left once the stronger claims have had theirs: the details panel.
// esc there is "back to the list".
func TestEscOnTheDetailsPanelReturnsFocusToTheList(t *testing.T) {
	m := homeWithGroups(t)

	rightPanel := constants.COMPONENT_BODY_DETAILS
	m = drive(m, collect(m.ChangeFocus(&rightPanel))...)

	if got := m.focusedComponent; got != rightPanel {
		t.Fatalf("precondition: expected the details panel focused, got %d", got)
	}

	updated, cmd := m.Update(escKey())
	m = updated.(AppModel)

	if got, want := m.focusedComponent, constants.COMPONENT_BODY_LIST; got != want {
		t.Errorf("esc left focus on component %d, want %d", got, want)
	}

	got, ok := focusedComponentFrom(collect(cmd))
	if !ok {
		t.Error("esc sent no focus message")
	} else if want := constants.COMPONENT_BODY_LIST; got != want {
		t.Errorf("esc focused component %d, want %d", got, want)
	}
}

// The ladder when a filter stands on a list that is not focused: esc cannot
// reach the filter, so it takes focus back to the list first; the next esc
// clears it.
func TestEscMovesFocusBeforeItClearsAnUnfocusedListsFilter(t *testing.T) {
	m := withFilterApplied(t, homeWithGroups(t))

	rightPanel := constants.COMPONENT_BODY_DETAILS
	m = drive(m, collect(m.ChangeFocus(&rightPanel))...)

	updated, cmd := m.Update(escKey())
	msgs := collect(cmd)

	if got, want := updated.(AppModel).focusedComponent, constants.COMPONENT_BODY_LIST; got != want {
		t.Errorf("first esc: focus got %d, want %d (the filter stands)", got, want)
	}
	if clearedFilter(msgs) {
		t.Error("first esc cleared a filter on a list it could not reach")
	}

	// Drive the messages the esc produced - the runtime would deliver them,
	// and the focus one is what lets the list see the next esc at all.
	m = drive(updated, msgs...)

	updated, cmd = m.Update(escKey())
	m = updated.(AppModel)

	if !clearedFilter(collect(cmd)) {
		t.Error("second esc did not clear the focused list's filter")
	}
}

// On an unfiltered list there is no details panel to come back from and no
// filter to clear, so esc is nobody's key.
func TestEscOnAnUnfilteredListDoesNothing(t *testing.T) {
	m := homeWithGroups(t)

	_, cmd := m.Update(escKey())

	for _, msg := range collect(cmd) {
		switch msg.(type) {
		case cmds.SetFocusMsg, cmds.SetListFilterStateMsg:
			t.Errorf("esc on an unfiltered list produced %T", msg)
		}
	}
}
