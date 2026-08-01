package components

import (
	"fmt"
	"image/color"
	"slices"
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/components/detailspanel"
	"github.com/filipemolina/stack-stitcher/src/components/groupdetailspanel"
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

// focusedDetailsPanel builds a detailspanel.Model sized and focused through
// its exported Update, the same messages AppModel sends it. Its own copy:
// detailspanel/Model_test.go's focusedDetails pokes the unexported fields
// directly, which this file can no longer do now that detailspanel is a
// separate package.
func focusedDetailsPanel(service types.ServiceConfig, width, height int) tea.Model {
	m := detailspanel.New(&service)
	m, _ = m.Update(cmds.SetBodyLayoutMsg{LeftWidth: 40, RightWidth: width, Height: height})
	m, _ = m.Update(cmds.SetFocusMsg(constants.COMPONENT_BODY_DETAILS))
	return m
}

// focusedGroupDetailsPanel builds a groupdetailspanel.Model selected, sized
// and focused through its exported Update, the same messages AppModel sends
// it. Its own copy of the same idea as focusedDetailsPanel: groupdetailspanel
// is a separate package now, so this file can no longer poke its unexported
// fields directly.
func focusedGroupDetailsPanel(services []types.ServiceConfig, selectedGroup string, width, height int) tea.Model {
	m := groupdetailspanel.New()
	for _, msg := range []tea.Msg{
		cmds.SetServicesListMsg(services),
		cmds.SetSelectedGroupMsg(selectedGroup),
		cmds.SetBodyLayoutMsg{LeftWidth: 40, RightWidth: width, Height: height},
		cmds.SetFocusMsg(constants.COMPONENT_BODY_DETAILS),
	} {
		m, _ = m.Update(msg)
	}
	return m
}

// The row used to paint all five buttons in the accent whatever the screen was
// doing, which promised that s/t/r/p/x were pressable when only a focused
// details panel answers them. The accent is the promise, so it is what this
// asserts on: present when the panel holds focus, gone when it does not.
func TestActionButtonsFollowFocus(t *testing.T) {
	for _, page := range []string{"Home", "Services"} {
		t.Run(page, func(t *testing.T) {
			focused := chrome.ActionButtons(80, appstyles.Active.BackgroundElevated, detailsContext(page, true))
			if !strings.Contains(focused, fgSGR(appstyles.Active.Accent)) {
				t.Error("a focused panel's action buttons are not drawn in the accent")
			}

			unfocused := chrome.ActionButtons(80, appstyles.Active.BackgroundPanel, detailsContext(page, false))
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

	row := chrome.ActionButtons(80, appstyles.Active.BackgroundElevated, ctx)
	if strings.Contains(row, fgSGR(appstyles.Active.Accent)) {
		t.Error("action buttons stay in the accent while an action is pending")
	}
}

// Dimming, not hiding: the row keeps its five buttons and its height in both
// states, so the panel body does not reflow every time Tab moves focus.
func TestActionButtonsKeepTheirShapeWhenDimmed(t *testing.T) {
	focused := chrome.ActionButtons(80, appstyles.Active.BackgroundElevated, detailsContext("Home", true))
	unfocused := chrome.ActionButtons(80, appstyles.Active.BackgroundPanel, detailsContext("Home", false))

	if got, want := ansi.Strip(unfocused), ansi.Strip(focused); got != want {
		t.Errorf("dimming changed the row's layout:\n got %q\nwant %q", got, want)
	}

	for _, label := range []string{"s Start", "t Stop", "r Restart", "p Pull", "x Remove", "l Logs"} {
		if !strings.Contains(ansi.Strip(unfocused), label) {
			t.Errorf("dimmed row is missing %q", label)
		}
	}
}

// The row is the panel's action set, so it has to be the whole set. It was a
// hand-written five that left out `l logs` while the footer offered it, which
// made the two disagree about what the panel can do.
func TestActionRowCoversEveryActionTheFooterOffers(t *testing.T) {
	for _, page := range []string{"Home", "Services"} {
		t.Run(page, func(t *testing.T) {
			ctx := detailsContext(page, true)

			inRow := make(map[string]bool)
			for _, button := range chrome.Buttons() {
				inRow[button.Binding.Help().Key] = true
			}

			for _, binding := range keys.Active(ctx) {
				help := binding.Help()

				// The footer also carries navigation and list keys; the row is
				// only about the panel's own verbs, which are the Details ones.
				if !isDetailsAction(binding) {
					continue
				}

				if !inRow[help.Key] {
					t.Errorf("footer offers %q (%s) on a focused details panel but the action row has no button for it", help.Key, help.Desc)
				}
			}
		})
	}
}

// isDetailsAction reports whether a binding is one of the details panel's own
// verbs, as opposed to a list or global key that happens to be live beside it.
// Edit and EditFile are excluded deliberately: they open an editor rather than
// act on the container, and the row is the container's controls.
func isDetailsAction(binding key.Binding) bool {
	for _, action := range []key.Binding{
		keys.Details.Start, keys.Details.Stop, keys.Details.Restart,
		keys.Details.Pull, keys.Details.Remove, keys.Details.Logs,
	} {
		if slices.Equal(action.Keys(), binding.Keys()) && action.Help() == binding.Help() {
			return true
		}
	}

	return false
}

// The destructive verb is not the peer of "restart", and the row says so.
func TestRemoveIsColoredAsDestructive(t *testing.T) {
	row := chrome.ActionButtons(80, appstyles.Active.BackgroundElevated, detailsContext("Home", true))

	if !strings.Contains(row, fgSGR(appstyles.Active.StatusError)) {
		t.Error("the remove button is not colored as destructive")
	}
}

// lipgloss wraps on the cell, not on the control, so a row wider than its panel
// used to break a button across two lines and push the panel's content out of
// its box. The row sheds whole buttons instead, lowest priority first, and what
// is shed is still on the footer and still pressable.
func TestActionRowShedsButtonsRatherThanWrapping(t *testing.T) {
	ctx := detailsContext("Home", true)

	full := chrome.ActionButtons(80, appstyles.Active.BackgroundElevated, ctx)
	if h := lipgloss.Height(full); h != 1 {
		t.Fatalf("a row that fits is %d rows tall, want 1", h)
	}

	// Every width from "nothing fits" up to the full row: none may wrap, none
	// may overflow the panel, and none may render a partial button.
	for width := 0; width <= lipgloss.Width(full); width++ {
		row := chrome.ActionButtons(width, appstyles.Active.BackgroundElevated, ctx)

		if h := lipgloss.Height(row); h != 1 {
			t.Errorf("width %d: row is %d rows tall, want 1", width, h)
		}
		if w := lipgloss.Width(row); w > max(width, 0) {
			t.Errorf("width %d: row rendered %d columns wide", width, w)
		}

		// A surviving button is whole: its key and its word, never a fragment.
		stripped := ansi.Strip(row)
		for _, button := range chrome.Buttons() {
			help := button.Binding.Help()
			if strings.Contains(stripped, help.Key+" "+chrome.ButtonLabel(help.Desc)[:1]) &&
				!strings.Contains(stripped, help.Key+" "+chrome.ButtonLabel(help.Desc)) {
				t.Errorf("width %d: %q is rendered as a fragment: %q", width, help.Desc, stripped)
			}
		}
	}
}

// The panels clip their body with MaxHeight, so the wrap never spilled the
// frame - it was absorbed, which is why it read as "the buttons look mangled"
// rather than "the layout is broken". A six-button row wrapped to thirty-one
// rows at the narrowest widths, and the clip ate the group's member table to
// pay for them.
//
// The wrap itself is guarded one test up, at the row, which is where it is
// observable. This is the standing guard that the frame stays inside the box
// AppModel gave it at any width - it would not have caught the wrap, and is
// not claimed to.
func TestNarrowPanelsStayInsideTheirBox(t *testing.T) {
	services := []types.ServiceConfig{
		{Name: "prowlarr", Image: "lscr.io/linuxserver/prowlarr:latest", Profiles: []string{"arr"}},
		{Name: "radarr", Image: "lscr.io/linuxserver/radarr:latest", Profiles: []string{"arr"}},
	}

	for _, width := range []int{100, 80, 60, 50, 40, 30, 24, 16, 10} {
		const height = 20

		group := focusedGroupDetailsPanel(services, "arr", width, height)
		service := focusedDetailsPanel(services[0], width, height)

		panels := map[string]tea.Model{"group": group, "service": service}
		for name, panel := range panels {
			frame := panel.View().Content

			if got := lipgloss.Height(frame); got != height {
				t.Errorf("%s panel at width %d: %d rows tall, want %d", name, width, got, height)
			}
			if got := lipgloss.Width(frame); got > width {
				t.Errorf("%s panel at width %d: %d columns wide, want ≤ %d", name, width, got, width)
			}
		}
	}
}

// Shedding follows the declared order, so a panel that can only hold three
// controls holds the three lifecycle verbs - not whichever happened to be
// listed first.
func TestActionRowShedsInPriorityOrder(t *testing.T) {
	ctx := detailsContext("Home", true)

	// Widths descend, so each step may only ever lose buttons.
	previous := map[string]bool{}
	first := true

	for width := lipgloss.Width(chrome.ActionButtons(80, appstyles.Active.BackgroundElevated, ctx)); width >= 0; width-- {
		stripped := ansi.Strip(chrome.ActionButtons(width, appstyles.Active.BackgroundElevated, ctx))

		present := map[string]bool{}
		for _, button := range chrome.Buttons() {
			help := button.Binding.Help()
			present[help.Desc] = strings.Contains(stripped, help.Key+" "+chrome.ButtonLabel(help.Desc))
		}

		if !first {
			for desc, was := range previous {
				if !was && present[desc] {
					t.Fatalf("width %d: %q came back after being shed", width, desc)
				}
			}
		}

		// remove is shed before pull, pull before logs, logs before restart.
		for _, pair := range [][2]string{{"remove", "pull"}, {"pull", "logs"}, {"logs", "restart"}, {"restart", "stop"}, {"stop", "start"}} {
			if present[pair[0]] && !present[pair[1]] {
				t.Errorf("width %d: %q survived while %q was shed, which inverts the drop order", width, pair[0], pair[1])
			}
		}

		previous, first = present, false
	}
}

// The labels and shortcuts are the bindings' own help text, so a rebound key
// moves the button with it rather than leaving it advertising the old one.
func TestActionButtonsRenderTheBindingsOwnHelp(t *testing.T) {
	row := ansi.Strip(chrome.ActionButtons(80, appstyles.Active.BackgroundElevated, detailsContext("Home", true)))

	for _, button := range chrome.Buttons() {
		help := button.Binding.Help()
		want := help.Key + " " + chrome.ButtonLabel(help.Desc)

		if !strings.Contains(row, want) {
			t.Errorf("action row does not render %q for binding %v", want, help)
		}
	}
}

// panelActionRowLine is the 0-based line of `screen` the action row sits on,
// found by the shortcut of the first button the row shows. -1 when the row is
// not on the screen at all.
func panelActionRowLine(screen string) int {
	want := keys.Details.Start.Help().Key + " " + chrome.ButtonLabel(keys.Details.Start.Help().Desc)

	for i, line := range strings.Split(screen, "\n") {
		if strings.Contains(ansi.Strip(line), want) {
			return i
		}
	}

	return -1
}

// Both panels are documented as pinning the action row to the bottom of their
// body ("The action row" in docs/DESIGN.md), and the group panel did it while
// the service panel let the row land wherever its tables happened to end -
// mid-panel, with a dozen blank rows under it. The two are one layout now
// (panelBodyWithActions), so this pins both to the same line: the last body
// row, which is the frame's bottom padding row minus one.
func TestDetailsPanelsPinActionRowToBottom(t *testing.T) {
	const height = 40

	service := focusedDetailsPanel(types.ServiceConfig{Name: "web", Image: "nginx:latest"}, 90, height)

	group := focusedGroupDetailsPanel(
		[]types.ServiceConfig{{Name: "web", Profiles: []string{"stack"}}},
		"stack", 90, height,
	)

	// The frame pads by one row at the bottom, so the last body row is the
	// second-to-last line of the panel.
	want := height - 2

	for name, screen := range map[string]string{
		"service": service.View().Content,
		"group":   group.View().Content,
	} {
		if got := panelActionRowLine(screen); got != want {
			t.Errorf("%s panel: action row on line %d of %d, want %d\n%s",
				name, got, lipgloss.Height(screen), want, ansi.Strip(screen))
		}
	}
}

// A panel too short for its content has to lose content, not actions: the row
// is the panel's floor, and a body that pushed it off the bottom would take
// the only visible affordance for start/stop with it.
func TestServiceDetailsKeepsActionRowWhenContentOverflows(t *testing.T) {
	m := focusedDetailsPanel(types.ServiceConfig{Name: "web", Image: "nginx:latest"}, 90, 12)

	screen := m.View().Content

	if got, want := panelActionRowLine(screen), 12-2; got != want {
		t.Errorf("action row on line %d, want %d\n%s", got, want, ansi.Strip(screen))
	}
	if got := lipgloss.Height(screen); got != 12 {
		t.Errorf("panel is %d rows tall, want 12", got)
	}
}
