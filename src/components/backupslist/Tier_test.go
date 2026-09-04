package backupslist

import (
	"image/color"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/components/chrome"
)

// bgSequence is the escape sequence lipgloss emits to set c as a background.
// Deriving it by rendering a probe rather than spelling out an SGR string
// keeps the test honest if lipgloss changes how it writes colours.
func bgSequence(t *testing.T, c color.Color) string {
	t.Helper()

	rendered := lipgloss.NewStyle().Background(c).Render("X")
	prefix, _, ok := strings.Cut(rendered, "X")
	if !ok || prefix == "" {
		t.Fatalf("could not read a background sequence out of %q", rendered)
	}

	return prefix
}

// Focus reaches the screen as a background tier, not as a border: the frame
// itself has to move, or a focused panel is indistinguishable from the one
// beside it.
//
// This asserts on a rendered frame, so it is negative-controlled by checking
// both tiers on the same panel: whichever one is missing, exactly one of the
// two assertions fires.
func TestFocusLiftsTheRenderedPanelATier(t *testing.T) {
	elevated := bgSequence(t, chrome.PanelBgFor(true))
	panel := bgSequence(t, chrome.PanelBgFor(false))

	if elevated == panel {
		t.Fatal("the two tiers render the same sequence; this test cannot tell them apart")
	}

	m := listWith(t, 8)

	focused := m.View().Content
	if !strings.Contains(focused, elevated) {
		t.Error("the focused list does not render on the elevated tier")
	}

	unfocused := focus(t, m, apptypes.BackupsPreview).View().Content
	if !strings.Contains(unfocused, panel) {
		t.Error("the unfocused list does not render on the panel tier")
	}
	if strings.Contains(unfocused, elevated) {
		t.Error("the unfocused list is still painting the elevated tier somewhere")
	}

	// The lift changes colour and nothing else: DESIGN.md's reason for using a
	// tier rather than a border is that the panel's box stays the same size,
	// so no row of the body is lost to gaining or losing focus.
	if got, want := ansi.Strip(unfocused), ansi.Strip(focused); got != want {
		t.Errorf("focus changed the panel's text, not just its tier:\n%s\nwant:\n%s", got, want)
	}
}
