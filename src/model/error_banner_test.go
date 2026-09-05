package model

import (
	"testing"
)

// Esc dismisses a foreground error banner - the errors that stay until the
// next successful foreground operation, which before this had no manual
// dismissal at all.
func TestEscDismissesAForegroundErrorBanner(t *testing.T) {
	m := withGroupsLoaded(t)
	m.lastError = "docker start failed"
	m.lastErrorFromPoll = false

	m = updateForTest(t, m, keyPress(teaKeyEsc()))

	if m.lastError != "" {
		t.Errorf("esc did not dismiss the banner: %q", m.lastError)
	}
	if m.lastErrorFromPoll {
		t.Error("esc left lastErrorFromPoll set")
	}
}

// Esc dismisses a poll-sourced banner too. A recovered poll already clears
// its own; this is the manual way off one that has not recovered.
func TestEscDismissesAPollErrorBanner(t *testing.T) {
	m := withGroupsLoaded(t)
	m.lastError = "docker daemon unavailable"
	m.lastErrorFromPoll = true

	m = updateForTest(t, m, keyPress(teaKeyEsc()))

	if m.lastError != "" {
		t.Errorf("esc did not dismiss the poll banner: %q", m.lastError)
	}
	if m.lastErrorFromPoll {
		t.Error("esc left lastErrorFromPoll set")
	}
}

// Esc dismisses the banner and leaves the selection alone. Dismissing is the
// last rung of the ladder now: there is no deselect step under it, because a
// populated list always has a row under the cursor and that row is the
// selection. A second esc therefore does nothing.
func TestEscDismissesTheBannerAndKeepsTheSelection(t *testing.T) {
	m := withGroupsLoaded(t)
	m.selection.groupName = "core"
	m.lastError = "boom"

	// First esc: banner goes, selection stays.
	m = updateForTest(t, m, keyPress(teaKeyEsc()))
	if m.lastError != "" {
		t.Errorf("first esc did not dismiss the banner: %q", m.lastError)
	}
	if m.selection.groupName != "core" {
		t.Errorf("first esc deslected the group: %q", m.selection.groupName)
	}

	// Second esc: nothing left for it to do, and in particular it must not
	// clear the selection.
	m = updateForTest(t, m, keyPress(teaKeyEsc()))
	if m.selection.groupName != "core" {
		t.Errorf("second esc cleared the selection: %q", m.selection.groupName)
	}
}

// Esc with no banner and no selection does nothing - the banner rung is
// skipped, and the deselect rung is a no-op with nothing selected. This is the
// baseline that proves the banner rung is what changed, not the deselect rung.
func TestEscWithNoBannerAndNoSelectionDoesNothing(t *testing.T) {
	m := withGroupsLoaded(t)
	// withGroupsLoaded starts with no selection.
	m.selection.groupName = ""

	m = updateForTest(t, m, keyPress(teaKeyEsc()))

	if m.selection.groupName != "" {
		t.Errorf("esc selected a group with nothing to deselect: %q", m.selection.groupName)
	}
	if m.lastError != "" {
		t.Errorf("esc surfaced a banner: %q", m.lastError)
	}
}
