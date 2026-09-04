package chrome

import (
	"testing"

	"github.com/filipemolina/cais/src/appstyles"
)

// Focus is shown by lifting the panel a background tier, not by a heavier
// border, so the panel's box is the same size either way. The two tiers have
// to actually differ or the lift is invisible - the contrast between them is
// asserted for every theme in appstyles.Contrast_test.
func TestPanelBgForLiftsAndDropsATier(t *testing.T) {
	if PanelBgFor(true) != appstyles.Active.BackgroundElevated {
		t.Error("a focused panel should sit on the elevated tier")
	}
	if PanelBgFor(false) != appstyles.Active.BackgroundPanel {
		t.Error("an unfocused panel should sit a tier below, on the panel tier")
	}
	if PanelBgFor(true) == PanelBgFor(false) {
		t.Error("focused and unfocused panels render identically; the lift is invisible")
	}
}

// The always-active panels keep the tier they have. PanelBg is what every
// other panel in the app calls, and focus being gone there is what made the
// elevated tier their steady state.
func TestPanelBgIsStillTheFocusedTier(t *testing.T) {
	if PanelBg() != PanelBgFor(true) {
		t.Error("PanelBg should be the tier a focused panel gets")
	}
}

// A list row that is not the cursor sits flush on whatever tier its panel is
// on. Rows are rendered and sealed one at a time, so a row that hardcoded the
// elevated tier would float above an unfocused panel's own frame.
func TestListRowBgFollowsThePanelTier(t *testing.T) {
	unfocused := PanelBgFor(false)

	if got := ListRowBgOn(false, unfocused); got != unfocused {
		t.Errorf("an inactive row on an unfocused panel renders on %v, want the panel's own tier %v",
			got, unfocused)
	}

	// The cursor row keeps its own register on either tier: ModalBg is not
	// derived from the panel tiers, so it reads as the cursor regardless.
	if ListRowBgOn(true, unfocused) != appstyles.Active.ModalBg {
		t.Error("the cursor row should keep ModalBg on an unfocused panel")
	}
	if ListRowBgOn(true, PanelBgFor(true)) != ListRowBgOn(true, unfocused) {
		t.Error("the cursor row's background should not move with focus")
	}
}

// ListRowBg is ListRowBgOn on the always-active tier, so the dozen panels that
// never lost focus keep the exact colours they had.
func TestListRowBgIsUnchangedForTheAlwaysActivePanels(t *testing.T) {
	for _, active := range []bool{true, false} {
		if ListRowBg(active) != ListRowBgOn(active, PanelBg()) {
			t.Errorf("ListRowBg(%v) drifted from ListRowBgOn(%v, PanelBg())", active, active)
		}
	}
}
