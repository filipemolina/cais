package components

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/components/detailspanel"
	"github.com/filipemolina/cais/src/components/groupdetailspanel"
)

// focusedDetailsPanel builds a detailspanel.Model sized and focused through
// its exported Update, the same messages AppModel sends it. Its own copy:
// detailspanel/Model_test.go's focusedDetails pokes the unexported fields
// directly, which this file can no longer do now that detailspanel is a
// separate package.
func focusedDetailsPanel(service types.ServiceConfig, width, height int, extra ...tea.Msg) tea.Model {
	m := detailspanel.New(&service, "10.0.0.5")
	msgs := append([]tea.Msg{
		cmds.SetBodyLayoutMsg{LeftWidth: 40, RightWidth: width, Height: height},
	}, extra...)

	for _, msg := range msgs {
		m, _ = m.Update(msg)
	}

	return m
}

// focusedGroupDetailsPanel builds a groupdetailspanel.Model selected, sized
// and focused through its exported Update, the same messages AppModel sends
// it. Its own copy of the same idea as focusedDetailsPanel: groupdetailspanel
// is a separate package now, so this file can no longer poke its unexported
// fields directly.
func focusedGroupDetailsPanel(services []types.ServiceConfig, selectedGroup string, width, height int, extra ...tea.Msg) tea.Model {
	m := groupdetailspanel.New()
	msgs := append([]tea.Msg{
		cmds.SetServicesListMsg(services),
		cmds.SetSelectedGroupMsg(selectedGroup),
		cmds.SetBodyLayoutMsg{LeftWidth: 40, RightWidth: width, Height: height},
	}, extra...)

	for _, msg := range msgs {
		m, _ = m.Update(msg)
	}

	return m
}

// The panels clip their body with MaxHeight, so nothing they render can spill
// the frame - it is absorbed by eating the content instead, which is why an
// overflow here reads as "the panel looks mangled" rather than "the layout is
// broken". This is the standing guard that the frame stays inside the box
// AppModel gave it at any width.
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

// lineOf is the 0-based line of `screen` that contains `want`, or -1.
func lineOf(screen, want string) int {
	for i, line := range strings.Split(screen, "\n") {
		if strings.Contains(ansi.Strip(line), want) {
			return i
		}
	}

	return -1
}

// pendingAction is the message AppModel sends a panel when a docker action
// starts, which is what puts the spinner in the panel's footer.
func pendingAction(target string, isGroup bool) tea.Msg {
	return cmds.SetPendingActionMsg{Action: "start", Target: target, IsGroup: isGroup}
}

// Both panels pin their footer to the bottom of their body ("The panel footer"
// in docs/DESIGN.md), and the group panel did it while the service panel let
// its footer land wherever its tables happened to end - mid-panel, with a dozen
// blank rows under it. The two are one layout now (PanelBodyWithFooter), so
// this pins both to the same line: the last body row, which is the frame's
// bottom padding row minus one.
func TestDetailsPanelsPinPendingActionToBottom(t *testing.T) {
	const height = 40

	service := focusedDetailsPanel(
		types.ServiceConfig{Name: "web", Image: "nginx:latest"}, 90, height,
		pendingAction("web", false),
	)

	group := focusedGroupDetailsPanel(
		[]types.ServiceConfig{{Name: "web", Profiles: []string{"stack"}}},
		"stack", 90, height,
		pendingAction("stack", true),
	)

	// The frame pads by one row at the bottom, so the last body row is the
	// second-to-last line of the panel.
	want := height - 2

	for name, tc := range map[string]struct {
		screen string
		desc   string
	}{
		"service": {service.View().Content, chrome.ActionDescription("start", "web", false)},
		"group":   {group.View().Content, chrome.ActionDescription("start", "stack", true)},
	} {
		if got := lineOf(tc.screen, tc.desc); got != want {
			t.Errorf("%s panel: pending action on line %d of %d, want %d\n%s",
				name, got, lipgloss.Height(tc.screen), want, ansi.Strip(tc.screen))
		}
	}
}

// A panel too short for its content has to lose content, not its footer: a body
// that pushed the spinner off the bottom would leave a running docker action
// with no feedback at all.
func TestServiceDetailsKeepsPendingActionWhenContentOverflows(t *testing.T) {
	m := focusedDetailsPanel(
		types.ServiceConfig{Name: "web", Image: "nginx:latest"}, 90, 12,
		pendingAction("web", false),
	)

	screen := m.View().Content

	if got, want := lineOf(screen, chrome.ActionDescription("start", "web", false)), 12-2; got != want {
		t.Errorf("pending action on line %d, want %d\n%s", got, want, ansi.Strip(screen))
	}
	if got := lipgloss.Height(screen); got != 12 {
		t.Errorf("panel is %d rows tall, want 12", got)
	}
}

// A stopped group's panel has no standing footer: the start hint is gone, so
// the body's last row is empty rather than advertising a key. The spinner is
// the only thing that ever claims the footer slot, and it still pins to the
// bottom (TestDetailsPanelsPinPendingActionToBottom).
func TestIdleGroupPanelSpendsNoRowOnItsFooter(t *testing.T) {
	const height = 40

	idle := focusedGroupDetailsPanel(
		[]types.ServiceConfig{{Name: "web", Profiles: []string{"stack"}}},
		"stack", 90, height,
	)

	lines := strings.Split(idle.View().Content, "\n")

	// The frame pads by one row at the bottom, so the last body row is the
	// second-to-last line of the panel. With no footer it is the background
	// gap filler - a row of spaces - where the hint used to sit.
	if got := strings.TrimSpace(ansi.Strip(lines[height-2])); got != "" {
		t.Errorf("idle group panel's last body row is %q, want empty - a standing hint would sit here\n%s",
			got, ansi.Strip(idle.View().Content))
	}
}

// An idle panel spends no rows on a footer it has nothing to put in. The
// service panel has no standing footer at all now, so its content runs to the
// bottom of the body rather than stopping one row short of it.
func TestIdleServicePanelSpendsNoRowOnItsFooter(t *testing.T) {
	const height = 40

	idle := focusedDetailsPanel(types.ServiceConfig{Name: "web", Image: "nginx:latest"}, 90, height)
	busy := focusedDetailsPanel(
		types.ServiceConfig{Name: "web", Image: "nginx:latest"}, 90, height,
		pendingAction("web", false),
	)

	// Same frame either way: the spinner replaces a body row rather than
	// growing the panel.
	if got, want := lipgloss.Height(idle.View().Content), lipgloss.Height(busy.View().Content); got != want {
		t.Errorf("idle panel is %d rows tall, busy panel %d - the footer changed the frame", got, want)
	}
}
