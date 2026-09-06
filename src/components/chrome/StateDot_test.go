package chrome

import (
	"image/color"
	"strings"
	"testing"

	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/apptypes"
)

// The reason the fault tier has its own glyph at all: the ordinary stop is
// already drawn in StatusError, so a dead container distinguished only by
// another red would be invisible to anyone reading quickly - or at all, to a
// reader who cannot separate two reds.
func TestOnlyAFaultChangesTheGlyph(t *testing.T) {
	fault := StateGlyph(apptypes.TierFault)

	for _, tier := range []apptypes.StateTier{
		apptypes.TierRunning,
		apptypes.TierTransitional,
		apptypes.TierInert,
		apptypes.TierStopped,
		apptypes.TierUnknown,
	} {
		if got := StateGlyph(tier); got == fault {
			t.Errorf("tier %d draws with the fault glyph %q", tier, fault)
		}
	}

	if fault == dotGlyph {
		t.Error("the fault glyph is the ordinary dot, so a dead container is told apart by colour alone")
	}
}

// A dead container must not render the same as a stopped one. This is the
// whole point of the tier split, so it is asserted on the rendered output
// rather than on the tier.
func TestADeadContainerDoesNotRenderLikeAStoppedOne(t *testing.T) {
	bg := appstyles.Active.PanelBg

	dead := StateDot("dead", bg)
	exited := StateDot("exited", bg)

	if dead == exited {
		t.Fatal("dead and exited render identically")
	}
	if !strings.Contains(dead, faultGlyph) {
		t.Errorf("the dead dot %q does not carry the fault glyph %q", dead, faultGlyph)
	}
	if !strings.Contains(exited, dotGlyph) {
		t.Errorf("the exited dot %q is not the ordinary dot", exited)
	}
}

// The tiers that are told apart by colour must actually differ, or the states
// are split in the model and merged again on screen. The fault tier is not in
// this list: it is told apart by its glyph, for the reason below.
func TestTheColourTiersAreDistinct(t *testing.T) {
	seen := map[color.Color]apptypes.StateTier{}

	for _, tier := range []apptypes.StateTier{
		apptypes.TierRunning,
		apptypes.TierTransitional,
		apptypes.TierInert,
		apptypes.TierStopped,
		apptypes.TierUnknown,
	} {
		c := StateColor(tier)
		if other, clash := seen[c]; clash {
			t.Errorf("tiers %d and %d both draw in %v", other, tier, c)
		}
		seen[c] = tier
	}
}

// Why the fault glyph is load-bearing rather than decoration.
//
// StatusError and Danger are the same colour in most of the registered themes
// - 9 of 14 when this was written - so a dead container distinguished only by
// its ink would be pixel-identical to an ordinary stopped one for most users.
// The glyph is what actually carries it, and this asserts that across every
// theme rather than only the one that happens to be active.
func TestDeadIsDistinguishableFromStoppedInEveryTheme(t *testing.T) {
	original := appstyles.Active.Name
	t.Cleanup(func() { appstyles.SetTheme(original) })

	for name := range appstyles.Themes {
		t.Run(name, func(t *testing.T) {
			if !appstyles.SetTheme(name) {
				t.Fatalf("could not apply theme %q", name)
			}

			bg := appstyles.Active.PanelBg
			if StateDot("dead", bg) == StateDot("exited", bg) {
				t.Error("a dead container renders identically to a stopped one")
			}
		})
	}
}

// The pill says docker's own word, so a paused service does not have to be
// read as stopped.
func TestStateLabelKeepsDockersWord(t *testing.T) {
	cases := map[string]string{
		"running":    "RUNNING",
		"restarting": "RESTARTING",
		"paused":     "PAUSED",
		"dead":       "DEAD",
		"exited":     "EXITED",

		// No container at all is what the panels have always called STOPPED.
		"": "STOPPED",
	}

	for state, want := range cases {
		if got := StateLabel(state); got != want {
			t.Errorf("StateLabel(%q) = %q, want %q", state, got, want)
		}
	}
}
