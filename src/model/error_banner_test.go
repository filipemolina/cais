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

// Esc dismisses the banner before it deselects the current group - the same
// one-key-one-job ladder a filtered list clears on. The first esc clears the
// banner; the second esc deselects the group (there is no panel to return
// focus to anymore).
func TestEscDismissesTheBannerBeforeDeselecting(t *testing.T) {
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

	// Second esc: no banner in the way, so esc deselects the group.
	m = updateForTest(t, m, keyPress(teaKeyEsc()))
	if m.selection.groupName != "" {
		t.Errorf("second esc did not deselect the group: %q", m.selection.groupName)
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
