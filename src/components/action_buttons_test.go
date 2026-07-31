package components

import (
	"fmt"
	"image/color"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/constants"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

// fgSGR is the truecolor foreground escape lipgloss emits for c, which is how
// these tests tell an accent-rimmed button from a dimmed one without asserting
// on the exact bytes of a whole rendered row.
func fgSGR(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("38;2;%d;%d;%d", r>>8, g>>8, b>>8)
}

// detailsContext is the screen state for a details panel with a subject
// selected, focused or not.
func detailsContext(page string, focused bool) keys.Context {
	component := constants.COMPONENT_BODY_LIST
	if focused {
		component = constants.COMPONENT_BODY_DETAILS
	}

	return keys.Context{Page: page, Focused: component, Selected: true}
}

// The row used to paint all five buttons in the accent whatever the screen was
// doing, which promised that s/t/r/p/x were pressable when only a focused
// details panel answers them. The accent is the promise, so it is what this
// asserts on: present when the panel holds focus, gone when it does not.
func TestActionButtonsFollowFocus(t *testing.T) {
	for _, page := range []string{"Home", "Services"} {
		t.Run(page, func(t *testing.T) {
			focused := renderActionButtons(80, appstyles.Active.BackgroundElevated, detailsContext(page, true))
			if !strings.Contains(focused, fgSGR(appstyles.Active.Accent)) {
				t.Error("a focused panel's action buttons are not drawn in the accent")
			}

			unfocused := renderActionButtons(80, appstyles.Active.BackgroundPanel, detailsContext(page, false))
			if strings.Contains(unfocused, fgSGR(appstyles.Active.Accent)) {
				t.Error("an unfocused panel's action buttons are drawn in the accent, which reads as pressable")
			}
			if !strings.Contains(unfocused, fgSGR(appstyles.Active.TextDim)) {
				t.Error("an unfocused panel's action buttons are not dimmed")
			}
		})
	}
}

// A pending action already disables the keys (keys.Active drops them), so the
// row has to agree. The panels swap a spinner in over the top of it today, but
// the row must not depend on that to be honest about itself.
func TestActionButtonsDimWhileAnActionIsPending(t *testing.T) {
	ctx := detailsContext("Home", true)
	ctx.PendingAction = true

	row := renderActionButtons(80, appstyles.Active.BackgroundElevated, ctx)
	if strings.Contains(row, fgSGR(appstyles.Active.Accent)) {
		t.Error("action buttons stay in the accent while an action is pending")
	}
}

// Dimming, not hiding: the row keeps its five buttons and its height in both
// states, so the panel body does not reflow every time Tab moves focus.
func TestActionButtonsKeepTheirShapeWhenDimmed(t *testing.T) {
	focused := renderActionButtons(80, appstyles.Active.BackgroundElevated, detailsContext("Home", true))
	unfocused := renderActionButtons(80, appstyles.Active.BackgroundPanel, detailsContext("Home", false))

	if got, want := ansi.Strip(unfocused), ansi.Strip(focused); got != want {
		t.Errorf("dimming changed the row's layout:\n got %q\nwant %q", got, want)
	}

	for _, label := range []string{"s Start", "t Stop", "r Restart", "p Pull", "x Remove"} {
		if !strings.Contains(ansi.Strip(unfocused), label) {
			t.Errorf("dimmed row is missing %q", label)
		}
	}
}

// The labels and shortcuts are the bindings' own help text, so a rebound key
// moves the button with it rather than leaving it advertising the old one.
func TestActionButtonsRenderTheBindingsOwnHelp(t *testing.T) {
	row := ansi.Strip(renderActionButtons(80, appstyles.Active.BackgroundElevated, detailsContext("Home", true)))

	for _, binding := range actionButtonKeys() {
		help := binding.Help()
		want := help.Key + " " + buttonLabel(help.Desc)

		if !strings.Contains(row, want) {
			t.Errorf("action row does not render %q for binding %v", want, help)
		}
	}
}
