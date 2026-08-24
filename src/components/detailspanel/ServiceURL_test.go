package detailspanel

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/compose-spec/compose-go/v2/types"
)

func withWebURL() types.ServiceConfig {
	return types.ServiceConfig{
		Name:  "navidrome",
		Image: "deluan/navidrome:latest",
		Ports: []types.ServicePortConfig{{Published: "14533", Target: 4533}},
	}
}

// The rendered panel contains an OSC 8 hyperlink escape sequence carrying
// the full URL, and the stripped output still shows it at the right width -
// see docs/plans/service-urls.md's TestWebRowIsAHyperlink.
func TestWebRowIsAHyperlink(t *testing.T) {
	svc := withWebURL()
	m := Model{service: &svc, host: "192.168.1.10", panelWidth: 100, panelHeight: 30}

	frame := m.renderConfigTable(80)

	if !strings.Contains(frame, "\x1b]8;;http://192.168.1.10:14533") {
		t.Errorf("config table does not carry the OSC 8 hyperlink:\n%q", frame)
	}
	if !strings.Contains(ansi.Strip(frame), "http://192.168.1.10:14533") {
		t.Errorf("stripped config table does not show the URL:\n%s", ansi.Strip(frame))
	}
}

// The D5 trap, pinned: chrome.Truncate must never cut through the middle of
// the hyperlink's escape sequence at any panel width. An "\x1b]8;" opening
// sequence always has a matching close in well-formed output; a truncation
// bug leaves one dangling.
func TestWebRowSurvivesNarrowPanel(t *testing.T) {
	svc := withWebURL()

	for width := 100; width >= 20; width-- {
		m := Model{service: &svc, host: "192.168.1.10", panelWidth: width, panelHeight: 30}
		frame := m.renderConfigTable(width)

		opens := strings.Count(frame, "\x1b]8;")
		if opens%2 != 0 {
			t.Fatalf("width %d: odd count (%d) of OSC 8 sequences - one is truncated:\n%q", width, opens, frame)
		}
	}
}

// A service with no published ports gets no Web row - the config table's
// existing "a service states only what it defines" rule.
func TestNoWebRowWithoutPorts(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Image: "nginx:alpine"}
	m := Model{service: &svc, host: "192.168.1.10"}

	rows := m.configRows(60)

	for _, row := range rows {
		if row.label == "Web" {
			t.Fatalf("Web row present for a service with no ports: %+v", row)
		}
	}
}

// y copies the resolved URL and sets the status-line confirmation.
func TestCopyURLKeySetsTheStatusMessage(t *testing.T) {
	svc := withWebURL()
	m := Model{
		service:     &svc,
		host:        "192.168.1.10",
		panelWidth:  100,
		panelHeight: 30,
	}

	updated, cmd := m.Update(keyPress('y'))
	got := updated.(Model)

	if got.urlMessage != "copied http://192.168.1.10:14533" {
		t.Errorf("urlMessage = %q, want the copied confirmation", got.urlMessage)
	}
	if cmd == nil {
		t.Fatal("y produced no command - tea.SetClipboard was not dispatched")
	}
}

// y on a service with no URL does nothing - there is nothing to copy.
func TestCopyURLKeyDoesNothingWithoutAURL(t *testing.T) {
	svc := types.ServiceConfig{Name: "web", Image: "nginx:alpine"}
	m := Model{service: &svc, panelWidth: 100, panelHeight: 30}

	updated, cmd := m.Update(keyPress('y'))
	got := updated.(Model)

	if got.urlMessage != "" {
		t.Errorf("urlMessage = %q, want empty with no URL to copy", got.urlMessage)
	}
	if cmd != nil {
		t.Error("expected no command with no URL to copy")
	}
}
