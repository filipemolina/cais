package model

import (
	"testing"

	"github.com/filipemolina/stack-stitcher/src/constants"
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

// Esc dismisses the banner before it backs out of the details panel - the
// same one-key-one-job ladder a filtered list clears on. The first esc clears
// the banner and leaves focus where it was; the second esc navigates back.
func TestEscDismissesTheBannerBeforeNavigatingBack(t *testing.T) {
	m := withGroupsLoaded(t)
	m.focusedComponent = constants.COMPONENT_BODY_DETAILS
	m.lastError = "boom"

	// First esc: banner goes, focus stays on the details panel.
	m = updateForTest(t, m, keyPress(teaKeyEsc()))
	if m.lastError != "" {
		t.Errorf("first esc did not dismiss the banner: %q", m.lastError)
	}
	if m.focusedComponent != constants.COMPONENT_BODY_DETAILS {
		t.Errorf("first esc moved focus to %d, want to stay on details (%d)",
			m.focusedComponent, constants.COMPONENT_BODY_DETAILS)
	}

	// Second esc: no banner in the way, so esc backs out to the list.
	m = updateForTest(t, m, keyPress(teaKeyEsc()))
	if m.focusedComponent != constants.COMPONENT_BODY_LIST {
		t.Errorf("second esc did not back out to the list: focus = %d, want %d",
			m.focusedComponent, constants.COMPONENT_BODY_LIST)
	}
}

// Esc with no banner and focus already on the list does nothing - the banner
// rung is skipped, and the back-to-list rung is a no-op on the list. This is
// the baseline that proves the banner rung is what changed, not the back rung.
func TestEscOnTheListWithNoBannerDoesNothing(t *testing.T) {
	m := withGroupsLoaded(t)
	// withGroupsLoaded starts on the list.
	if m.focusedComponent != constants.COMPONENT_BODY_LIST {
		t.Fatalf("precondition: focus = %d, want the list (%d)",
			m.focusedComponent, constants.COMPONENT_BODY_LIST)
	}

	m = updateForTest(t, m, keyPress(teaKeyEsc()))

	if m.focusedComponent != constants.COMPONENT_BODY_LIST {
		t.Errorf("esc moved focus off the list with no banner: focus = %d",
			m.focusedComponent)
	}
}
