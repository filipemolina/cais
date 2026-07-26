package components

import (
	"strings"
	"testing"

	"github.com/filipemolina/stack-stitcher/src/constants"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

// joinHints renders hints the way the bar reads them out, so a failure prints
// the whole footer rather than a struct dump.
func joinHints(hints []KeyHint) string {
	parts := make([]string, 0, len(hints))
	for _, hint := range hints {
		parts = append(parts, hint.Key+" "+hint.Desc)
	}

	return strings.Join(parts, " · ")
}

// TestFooterHints pins the footer to what it advertised before the bindings
// moved into src/keys. Every expectation here was transcribed from the
// hand-written table this method used to hold: the point of the refactor was
// that the footer keeps saying exactly the same thing while the source of the
// truth changes underneath it.
func TestFooterHints(t *testing.T) {
	tests := []struct {
		name  string
		model KeybindingBarModel
		want  string
	}{
		{
			name:  "groups list with groups",
			model: KeybindingBarModel{activePage: "Home", focusedComponent: constants.COMPONENT_BODY_LIST},
			want:  "space select · n new · d delete · ↑/↓ navigate · tab next",
		},
		{
			name:  "groups list while empty",
			model: KeybindingBarModel{activePage: "Home", focusedComponent: constants.COMPONENT_BODY_LIST, groupsListEmpty: true},
			want:  "n new · ↑/↓ navigate · tab next",
		},
		{
			name:  "group details with nothing selected",
			model: KeybindingBarModel{activePage: "Home", focusedComponent: constants.COMPONENT_BODY_DETAILS},
			want:  "tab next",
		},
		{
			name:  "group details with a group selected",
			model: KeybindingBarModel{activePage: "Home", focusedComponent: constants.COMPONENT_BODY_DETAILS, selectedGroup: "core"},
			want:  "s start · t stop · r restart · p pull · x remove · l logs · tab next",
		},
		{
			name:  "services list with services",
			model: KeybindingBarModel{activePage: "Services", focusedComponent: constants.COMPONENT_BODY_LIST},
			want:  "space select · ↑/↓ navigate · tab next",
		},
		{
			name:  "services list while empty",
			model: KeybindingBarModel{activePage: "Services", focusedComponent: constants.COMPONENT_BODY_LIST, servicesListEmpty: true},
			want:  "↑/↓ navigate · tab next",
		},
		{
			name:  "service details with nothing selected",
			model: KeybindingBarModel{activePage: "Services", focusedComponent: constants.COMPONENT_BODY_DETAILS},
			want:  "tab next",
		},
		{
			name:  "service details with a service selected",
			model: KeybindingBarModel{activePage: "Services", focusedComponent: constants.COMPONENT_BODY_DETAILS, selectedService: true},
			want:  "s start · t stop · r restart · p pull · x remove · l logs · e edit · E file · tab next",
		},
		{
			name:  "a page with nothing focusable offers no keys",
			model: KeybindingBarModel{activePage: "Compose Files", focusedComponent: constants.COMPONENT_BODY_LIST},
			want:  "",
		},
		{
			name:  "an unknown page still offers the focus ring",
			model: KeybindingBarModel{activePage: "Nowhere", focusedComponent: constants.COMPONENT_BODY_LIST},
			want:  "tab next",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := joinHints(tc.model.hintsFor()); got != tc.want {
				t.Errorf("footer hints\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}

// The right-hand side of the bar is fixed rather than context-dependent, so it
// gets its own expectation.
func TestFooterGlobalHints(t *testing.T) {
	want := "alt+· page · q quit"

	if got := joinHints(hintsFrom(keys.Globals())); got != want {
		t.Errorf("global hints\n got: %s\nwant: %s", got, want)
	}
}
