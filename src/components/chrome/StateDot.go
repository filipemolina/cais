package chrome

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/apptypes"
)

// The glyphs a state dot draws with.
//
// Every tier but one uses the same filled circle and is told apart by colour.
// TierFault does not, and deliberately: the ordinary stop (exited) is already
// drawn in StatusError, so a dead container drawn in another red would be a
// fault the user cannot pick out from the most routine state there is. The
// cross carries it at a glance and survives a terminal that renders the two
// reds close together - or a reader who cannot separate them at all.
//
// The groups list already varies its glyph to carry meaning (a full circle for
// all-running, a half circle for mixed), so this is the app's existing
// language rather than a new one.
const (
	dotGlyph   = "●"
	faultGlyph = "✕"
)

// StateColor returns the colour a tier draws in.
//
// TierStopped keeps StatusError rather than StatusStopped's grey: at the size
// of a single glyph the grey did not carry - see the note in
// serviceslist.statusDot, which is the failure that put it there.
func StateColor(tier apptypes.StateTier) color.Color {
	switch tier {
	case apptypes.TierRunning:
		return appstyles.Active.StatusRunning
	case apptypes.TierTransitional:
		return appstyles.Active.StatusStarting
	case apptypes.TierInert:
		return appstyles.Active.TextMuted
	case apptypes.TierFault:
		return appstyles.Active.Danger
	case apptypes.TierStopped:
		return appstyles.Active.StatusError
	default:
		// TierUnknown: docker has not answered yet, or cannot be reached.
		// Dim rather than red, because a page of red dots on arrival reads
		// as a page of dead services rather than as a page of unanswered
		// questions.
		return appstyles.Active.TextDim
	}
}

// StateGlyph returns the character a tier draws with.
func StateGlyph(tier apptypes.StateTier) string {
	if tier == apptypes.TierFault {
		return faultGlyph
	}

	return dotGlyph
}

// StateDot renders the status glyph for a container state, on bg so it stays
// legible across a row's selection and focus states.
func StateDot(state string, bg color.Color) string {
	tier := apptypes.ClassifyContainerState(state)

	return lipgloss.NewStyle().
		Foreground(StateColor(tier)).
		Background(bg).
		Render(StateGlyph(tier))
}

// StateLabel returns the pill text for a container state: docker's own word,
// upper-cased, so a service that is paused says PAUSED rather than being
// flattened into STOPPED.
//
// An absent container is STOPPED, which is what "no container" has always
// meant to the panels; an empty state only reaches here once docker has
// answered.
func StateLabel(state string) string {
	if strings.TrimSpace(state) == "" {
		return "STOPPED"
	}

	return strings.ToUpper(strings.TrimSpace(state))
}
