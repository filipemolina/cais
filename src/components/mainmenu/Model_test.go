package mainmenu

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/filipemolina/stack-stitcher/src/constants"
)

// navAt renders the nav bar at a terminal width, stripped of styling.
func navAt(t *testing.T, width int) string {
	t.Helper()

	nav, _ := New().Update(tea.WindowSizeMsg{Width: width, Height: 24})

	return ansi.Strip(nav.View().Content)
}

// The version is what a bug report needs and what tells two installs apart,
// so a terminal with room for it shows it.
func TestNavBarShowsTheVersion(t *testing.T) {
	bar := navAt(t, 120)

	if !strings.Contains(bar, constants.Version()) {
		t.Errorf("nav bar %q does not carry version %q", bar, constants.Version())
	}
}

// Narrow, it goes rather than pushing the wordmark off the row or wrapping
// the bar onto a second line - the tabs are what the nav is for.
//
// 64 columns is the interesting width: the tabs and the wordmark still fit,
// so the only thing that can overflow the row is the version. (Below about
// 58 the nav overflows on its own, which predates the version and is tracked
// in TODO.md alongside the footer's wrapping.)
func TestNavBarDropsTheVersionWhenNarrow(t *testing.T) {
	bar := navAt(t, 64)

	if strings.Contains(bar, constants.Version()) {
		t.Errorf("nav bar %q kept the version at 64 columns", bar)
	}
	if !strings.Contains(bar, "Groups") {
		t.Errorf("nav bar %q dropped a tab instead", bar)
	}
	if lines := strings.Count(bar, "\n"); lines > 0 {
		t.Errorf("nav bar wrapped onto %d extra lines: %q", lines, bar)
	}
}
