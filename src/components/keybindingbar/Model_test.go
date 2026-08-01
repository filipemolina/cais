package keybindingbar

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/constants"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

// joinHints renders hints the way the bar reads them out, so a failure prints
// the whole footer rather than a struct dump.
func joinHints(hints []chrome.KeyHint) string {
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
		model Model
		want  string
	}{
		{
			name:  "groups list with groups",
			model: Model{activePage: "Home", focusedComponent: constants.COMPONENT_BODY_LIST},
			want:  "space start · n new · e edit · d delete · R rename · / filter · ↑/↓ navigate · tab next",
		},
		{
			// The list has the keyboard: every other key is a letter.
			name:  "groups list while a filter is being typed",
			model: Model{activePage: "Home", focusedComponent: constants.COMPONENT_BODY_LIST, filterState: list.Filtering},
			want:  "enter apply · esc cancel",
		},
		{
			// The filter slot becomes the way out of the filter.
			name:  "groups list with a filter applied",
			model: Model{activePage: "Home", focusedComponent: constants.COMPONENT_BODY_LIST, filterState: list.FilterApplied},
			want:  "space start · n new · e edit · d delete · R rename · esc clear filter · ↑/↓ navigate · tab next",
		},
		{
			name:  "groups list while empty",
			model: Model{activePage: "Home", focusedComponent: constants.COMPONENT_BODY_LIST, groupsListEmpty: true},
			want:  "n new · ↑/↓ navigate · tab next",
		},
		{
			name:  "group details with nothing selected",
			model: Model{activePage: "Home", focusedComponent: constants.COMPONENT_BODY_DETAILS},
			want:  "n new · esc back · tab next",
		},
		{
			name:  "group details with a group selected",
			model: Model{activePage: "Home", focusedComponent: constants.COMPONENT_BODY_DETAILS, selectedGroup: "core"},
			want:  "n new · s start · t stop · r restart · p pull · x remove · l logs · esc back · tab next",
		},
		{
			name:  "services list with services",
			model: Model{activePage: "Services", focusedComponent: constants.COMPONENT_BODY_LIST},
			want:  "space start · / filter · ↑/↓ navigate · tab next",
		},
		{
			name:  "services list while a filter is being typed",
			model: Model{activePage: "Services", focusedComponent: constants.COMPONENT_BODY_LIST, filterState: list.Filtering},
			want:  "enter apply · esc cancel",
		},
		{
			name:  "services list while empty",
			model: Model{activePage: "Services", focusedComponent: constants.COMPONENT_BODY_LIST, servicesListEmpty: true},
			want:  "↑/↓ navigate · tab next",
		},
		{
			name:  "service details with nothing selected",
			model: Model{activePage: "Services", focusedComponent: constants.COMPONENT_BODY_DETAILS},
			want:  "esc back · tab next",
		},
		{
			name:  "service details with a service selected",
			model: Model{activePage: "Services", focusedComponent: constants.COMPONENT_BODY_DETAILS, selectedService: true},
			want:  "s start · t stop · r restart · p pull · x remove · l logs · y copy url · h healthcheck · e edit · E file · esc back · tab next",
		},
		{
			name:  "service details while inline editing",
			model: Model{activePage: "Services", focusedComponent: constants.COMPONENT_BODY_DETAILS, selectedService: true, editing: true},
			want:  "ctrl+s save · ctrl+o editor · tab indent · shift+tab outdent · esc back",
		},
		{
			name:  "the files page offers edit, browse and scroll",
			model: Model{activePage: "Compose Files", focusedComponent: constants.COMPONENT_BODY_LIST},
			want:  "E file · b browse · ↑/↓ scroll",
		},
		{
			name:  "an unknown page still offers the focus ring",
			model: Model{activePage: "Nowhere", focusedComponent: constants.COMPONENT_BODY_LIST},
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
			model := Model{composeFile: tc.file, composeFileOthers: tc.others}

			if got := ansi.Strip(model.composeFileSegment(tc.spare)); got != tc.want {
				t.Errorf("compose file segment at spare=%d\n got: %q\nwant: %q", tc.spare, got, tc.want)
			}
		})
	}
}

// The file name arrives by broadcast, the same way every other piece of state
// the bar shows does.
func TestFooterTakesTheComposeFileFromTheBroadcast(t *testing.T) {
	var model tea.Model = New()
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

		var model tea.Model = New()
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

// barAt renders the footer for a populated Home page at the given width, which
// is the context that overflows first: it advertises the most keys.
func barAt(t *testing.T, width int) string {
	t.Helper()

	var model tea.Model = New()
	for _, msg := range []tea.Msg{
		cmds.SetComposeFileMsg{Name: "/srv/homelab/compose.yaml"},
		cmds.SetGroupsListMsg([]string{"media", "infra", "downloads"}),
		tea.WindowSizeMsg{Width: width, Height: 24},
	} {
		model, _ = model.Update(msg)
	}

	return model.View().Content
}

// The bar is one line. It was not: the context hints plus the globals exceeded
// a normal terminal's width - measured at ~133 columns, not the "below 60" the
// TODO originally guessed - and lipgloss wrapped them onto a second and third
// line, each one eating a row of the body above. This is the guard that no
// width wraps it, including widths too narrow for the keys it refuses to shed.
// The floor is 20 rather than 0 because the bar's own 4 columns of padding are
// a lipgloss floor, not a shedding decision: below about 6 columns Width()
// cannot render narrower than the padding it was given, and no terminal is that
// size.
func TestFooterNeverWraps(t *testing.T) {
	for width := 200; width >= 20; width-- {
		bar := barAt(t, width)

		if got := lipgloss.Height(bar); got != 1 {
			t.Errorf("at width %d the bar is %d lines tall:\n%s", width, got, ansi.Strip(bar))
		}
		if got := lipgloss.Width(bar); got != width {
			t.Errorf("at width %d the bar rendered %d columns wide", width, got)
		}
	}
}

// Shedding is only safe because the way to everything shed survives it: `?
// help` opens the overlay that lists every binding, and `q quit` is the escape
// hatch. Both are ranked never-shed, so they are the last things on the bar.
func TestFooterKeepsHelpAndQuitAtEveryWidth(t *testing.T) {
	// 24 columns fits "? help · q quit" plus the bar's own padding; below that
	// the MaxHeight clip takes over and there is no honest answer.
	for width := 200; width >= 24; width-- {
		bar := ansi.Strip(barAt(t, width))

		for _, want := range []string{"? help", "q quit"} {
			if !strings.Contains(bar, want) {
				t.Errorf("at width %d the bar dropped %q: %q", width, want, bar)
			}
		}
	}
}

// The order is declared in keys.Priority and is deliberately not the display
// order. `1-3 page` goes first because the nav bar prints the digits already;
// tab and the arrows next because they are what a user tries unprompted; the
// page's own verbs last, rightmost first.
func TestFooterShedsInPriorityOrder(t *testing.T) {
	// Widths descend, so each step may only ever lose hints.
	previous := map[string]bool{}
	first := true

	for width := 200; width >= 24; width-- {
		bar := ansi.Strip(barAt(t, width))

		present := map[string]bool{}
		for _, hint := range []string{
			"1-3 page", "tab next", "↑/↓ navigate",
			"/ filter", "R rename", "d delete", "e edit", "n new", "space start",
		} {
			present[hint] = strings.Contains(bar, hint)
		}

		if !first {
			for hint, was := range previous {
				if !was && present[hint] {
					t.Fatalf("at width %d %q came back after being shed:\n%s", width, hint, bar)
				}
			}
		}

		// Each pair is (shed earlier, shed later): the left one must never
		// outlive the right one.
		for _, pair := range [][2]string{
			{"1-3 page", "tab next"},
			{"tab next", "↑/↓ navigate"},
			{"↑/↓ navigate", "/ filter"},
			{"/ filter", "R rename"},
			{"R rename", "d delete"},
			{"d delete", "e edit"},
			{"e edit", "n new"},
			{"n new", "space start"},
		} {
			if present[pair[0]] && !present[pair[1]] {
				t.Errorf("at width %d %q survived while %q was shed, which inverts the drop order:\n%s",
					width, pair[0], pair[1], bar)
			}
		}

		previous, first = present, false
	}
}

// A hint is whole or absent, never a fragment: the bar sheds controls, not
// columns, which is the whole difference between this and letting lipgloss wrap
// or truncate.
func TestFooterShedsWholeHints(t *testing.T) {
	for width := 200; width >= 24; width-- {
		bar := ansi.Strip(barAt(t, width))

		for _, hint := range []string{"space start", "n new", "e edit", "d delete", "R rename", "↑/↓ navigate", "tab next"} {
			key, desc, _ := strings.Cut(hint, " ")

			// The key without its word means the word was cut off mid-hint.
			if strings.Contains(bar, key+" "+desc[:1]) && !strings.Contains(bar, hint) {
				t.Errorf("at width %d %q is rendered as a fragment: %q", width, hint, bar)
			}
		}
	}
}
