package model

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/compose-spec/compose-go/v2/types"

	"github.com/filipemolina/cais/src/cmds"
)

// filterProject has services whose images are distinct, so the details panel
// can be identified by which image it is showing, and whose names let a
// one-letter filter hide the one the cursor starts on.
func filterProject() *types.Project {
	return &types.Project{
		Services: types.Services{
			"alpha":    types.ServiceConfig{Name: "alpha", Image: "nginx:alpine", Profiles: []string{"core"}},
			"bravo":    types.ServiceConfig{Name: "bravo", Image: "redis:7", Profiles: []string{"core"}},
			"worker":   types.ServiceConfig{Name: "worker", Image: "busybox", Profiles: []string{"core"}},
			"webproxy": types.ServiceConfig{Name: "webproxy", Image: "traefik", Profiles: []string{"core"}},
		},
	}
}

// pump feeds messages into the model and keeps feeding back what the
// resulting commands produced, standing in for the runtime's command
// round-trip. drive() alone discards commands, which hides everything
// downstream of list.FilterMatchesMsg - the message that populates the
// filtered rows, and so the one every filter assertion depends on.
func pump(t *testing.T, m AppModel, msgs ...tea.Msg) AppModel {
	t.Helper()

	queue := append([]tea.Msg{}, msgs...)
	for range 500 {
		if len(queue) == 0 {
			return m
		}

		msg := queue[0]
		queue = queue[1:]

		updated, cmd := m.Update(msg)
		m = updated.(AppModel)

		for _, produced := range collect(cmd) {
			if produced == nil {
				continue
			}
			// Container polling would re-enter indefinitely; the filter
			// path does not depend on it.
			switch produced.(type) {
			case cmds.GetRunningContainersMsg, cmds.GetContainerStatsMsg:
				continue
			}
			queue = append(queue, produced)
		}
	}

	t.Fatal("messages never settled")
	return m
}

// withServicesFiltered is the app on the Services page, showing filterProject,
// with term typed into the services list's filter and accepted.
func withServicesFiltered(t *testing.T, term string) AppModel {
	t.Helper()

	m := startup(120, 40)
	m = pump(t, m, cmds.GetConfigMsg{FileName: "compose.yaml", Project: filterProject()})
	m = applyLayout(m)
	m = pump(t, m, cmds.SetActivePageMsg("Services"))

	msgs := []tea.Msg{tea.KeyPressMsg{Code: '/', Text: "/"}}
	for _, r := range term {
		msgs = append(msgs, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	msgs = append(msgs, tea.KeyPressMsg{Code: tea.KeyEnter})

	return pump(t, m, msgs...)
}

// detailsText is what the Services page's details panel is currently drawing.
func detailsText(t *testing.T, m AppModel) string {
	t.Helper()

	return ansi.Strip(detailsPanel(t, m).View().Content)
}

// Regression test, end to end: filtering the services list must move the
// details panel to the service the filter put under the cursor.
//
// The cursor starts on alpha, which "w" hides. The selection used to be
// published only when list.Index() changed, and accepting a filter sends the
// cursor to the top - so a cursor already at the top kept index 0 while row 0
// became a different service, nothing was published, and the details panel
// went on describing a service that was no longer on screen. See
// serviceslist.Model.Update.
func TestFilteringMovesTheDetailsPanel(t *testing.T) {
	m := withServicesFiltered(t, "worker")

	details := detailsText(t, m)

	if !strings.Contains(details, "busybox") {
		t.Errorf("details panel did not move to worker after filtering. Panel:\n%s", details)
	}
	if strings.Contains(details, "nginx:alpine") {
		t.Errorf("details panel still describes alpha, which the filter hid. Panel:\n%s", details)
	}
}

// And it keeps following the cursor once the filter is applied.
func TestTheDetailsPanelFollowsTheCursorInsideAFilter(t *testing.T) {
	m := withServicesFiltered(t, "web")

	m = pump(t, m, tea.KeyPressMsg{Code: tea.KeyDown})

	details := detailsText(t, m)

	if !strings.Contains(details, "traefik") {
		t.Errorf("details panel did not follow the cursor to webproxy. Panel:\n%s", details)
	}
}
