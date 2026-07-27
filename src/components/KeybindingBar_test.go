package components

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/filipemolina/stack-stitcher/src/cmds"
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
			want:  "space select · n new · d delete · / filter · ↑/↓ navigate · tab next",
		},
		{
			// The list has the keyboard: every other key is a letter.
			name:  "groups list while a filter is being typed",
			model: KeybindingBarModel{activePage: "Home", focusedComponent: constants.COMPONENT_BODY_LIST, filterState: list.Filtering},
			want:  "enter apply · esc cancel",
		},
		{
			// The filter slot becomes the way out of the filter.
			name:  "groups list with a filter applied",
			model: KeybindingBarModel{activePage: "Home", focusedComponent: constants.COMPONENT_BODY_LIST, filterState: list.FilterApplied},
			want:  "space select · n new · d delete · esc clear filter · ↑/↓ navigate · tab next",
		},
		{
			name:  "groups list while empty",
			model: KeybindingBarModel{activePage: "Home", focusedComponent: constants.COMPONENT_BODY_LIST, groupsListEmpty: true},
			want:  "n new · ↑/↓ navigate · tab next",
		},
		{
			name:  "group details with nothing selected",
			model: KeybindingBarModel{activePage: "Home", focusedComponent: constants.COMPONENT_BODY_DETAILS},
			want:  "esc back · tab next",
		},
		{
			name:  "group details with a group selected",
			model: KeybindingBarModel{activePage: "Home", focusedComponent: constants.COMPONENT_BODY_DETAILS, selectedGroup: "core"},
			want:  "s start · t stop · r restart · p pull · x remove · l logs · esc back · tab next",
		},
		{
			name:  "services list with services",
			model: KeybindingBarModel{activePage: "Services", focusedComponent: constants.COMPONENT_BODY_LIST},
			want:  "space select · / filter · ↑/↓ navigate · tab next",
		},
		{
			name:  "services list while a filter is being typed",
			model: KeybindingBarModel{activePage: "Services", focusedComponent: constants.COMPONENT_BODY_LIST, filterState: list.Filtering},
			want:  "enter apply · esc cancel",
		},
		{
			name:  "services list while empty",
			model: KeybindingBarModel{activePage: "Services", focusedComponent: constants.COMPONENT_BODY_LIST, servicesListEmpty: true},
			want:  "↑/↓ navigate · tab next",
		},
		{
			name:  "service details with nothing selected",
			model: KeybindingBarModel{activePage: "Services", focusedComponent: constants.COMPONENT_BODY_DETAILS},
			want:  "esc back · tab next",
		},
		{
			name:  "service details with a service selected",
			model: KeybindingBarModel{activePage: "Services", focusedComponent: constants.COMPONENT_BODY_DETAILS, selectedService: true},
			want:  "s start · t stop · r restart · p pull · x remove · l logs · e edit · E file · esc back · tab next",
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

// TestFooterComposeFile pins the degradation ladder. The name is worth less
// than the keys next to it, so as the room runs out it gives up detail and
// then gives up entirely, rather than pushing the keys off the bar.
func TestFooterComposeFile(t *testing.T) {
	const path = "/srv/homelab/compose.yaml"

	tests := []struct {
		name   string
		file   string
		others int
		spare  int
		want   string
	}{
		{
			name:  "the full path when it fits",
			file:  path,
			spare: 40,
			want:  path + " · ",
		},
		{
			// The count is part of the answer to "which file?", so it rides
			// along through the ladder rather than being shed as detail.
			name:   "losing candidates are counted, and survive the basename",
			file:   path,
			others: 2,
			spare:  20,
			want:   "compose.yaml +2 · ",
		},
		{
			name:  "the basename when only that fits",
			file:  path,
			spare: 20,
			want:  "compose.yaml · ",
		},
		{
			name:  "nothing when even the basename does not fit",
			file:  path,
			spare: 8,
			want:  "",
		},
		{
			name:  "nothing when there is no room at all",
			file:  path,
			spare: -3,
			want:  "",
		},
		{
			name:  "a bare file name is already its own basename",
			file:  "docker-compose.yml",
			spare: 25,
			want:  "docker-compose.yml · ",
		},
		{
			name:  "no file loaded says so",
			file:  "",
			spare: 25,
			want:  "no compose file · ",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			model := KeybindingBarModel{composeFile: tc.file, composeFileOthers: tc.others}

			if got := ansi.Strip(model.composeFileSegment(tc.spare)); got != tc.want {
				t.Errorf("compose file segment at spare=%d\n got: %q\nwant: %q", tc.spare, got, tc.want)
			}
		})
	}
}

// The file name arrives by broadcast, the same way every other piece of state
// the bar shows does.
func TestFooterTakesTheComposeFileFromTheBroadcast(t *testing.T) {
	var model tea.Model = KeybindingBar()
	model, _ = model.Update(cmds.SetComposeFileMsg{
		Name:   "compose.yaml",
		Others: []string{"compose.yml", "docker-compose.yml"},
	})

	bar := ansi.Strip(model.View().Content)
	if !strings.Contains(bar, "compose.yaml") {
		t.Errorf("footer does not name the compose file: %q", bar)
	}
	if !strings.Contains(bar, "+2") {
		t.Errorf("footer does not count the losing candidates: %q", bar)
	}
}

// The reason the name degrades at all: it is an addition to a bar that was
// already laid out, so it must never cost the bar a line or a key. Each width
// is measured against the same bar without a file loaded - the bar wraps on its
// own once the two hint groups outgrow the terminal, which is a separate
// problem, so the assertion is that the name makes it no worse.
func TestFooterComposeFileNeverCrowdsOutTheKeys(t *testing.T) {
	renderAt := func(t *testing.T, width int, file string) string {
		t.Helper()

		var model tea.Model = KeybindingBar()
		if file != "" {
			model, _ = model.Update(cmds.SetComposeFileMsg{Name: file})
		}
		model, _ = model.Update(tea.WindowSizeMsg{Width: width, Height: 24})

		return model.View().Content
	}

	for _, width := range []int{200, 120, 80, 60, 40, 30, 20} {
		withFile := renderAt(t, width, "/srv/homelab/compose.yaml")
		baseline := renderAt(t, width, "")

		gotLines := strings.Count(withFile, "\n")
		wantLines := strings.Count(baseline, "\n")

		if gotLines > wantLines {
			t.Errorf("at width %d the file name cost the bar a line:\n got: %q\nwant no worse than: %q",
				width, ansi.Strip(withFile), ansi.Strip(baseline))
		}
		if got := lipgloss.Width(withFile); got != width {
			t.Errorf("bar at width %d rendered %d columns wide", width, got)
		}
		if bar := ansi.Strip(withFile); !strings.Contains(bar, "quit") {
			t.Errorf("bar at width %d dropped the global keys: %q", width, bar)
		}
	}
}

// The right-hand side of the bar is fixed rather than context-dependent, so it
// gets its own expectation.
func TestFooterGlobalHints(t *testing.T) {
	want := "1-3 page · ? help · q quit"

	if got := joinHints(hintsFrom(keys.Globals())); got != want {
		t.Errorf("global hints\n got: %s\nwant: %s", got, want)
	}
}
