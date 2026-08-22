package groupdetailspanel

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/cais/src/apptypes"
)

// Selecting the reserved ungrouped row renders the member table over the
// untagged services - the same table every other group gets, with a real
// header card and working action keys.
func TestSelectingUngroupedRendersTheUntaggedServices(t *testing.T) {
	m := panelWith([]types.ServiceConfig{
		{Name: "web", Image: "nginx:alpine", Profiles: []string{"core"}},
		{Name: "proxy", Image: "traefik:latest"},
		{Name: "cache", Image: "redis:7"},
	}, nil)
	m.selectedGroup = apptypes.UngroupedGroup

	body := ansi.Strip(m.renderBody())

	for _, name := range []string{"proxy", "cache"} {
		if !strings.Contains(body, name) {
			t.Errorf("untagged service %q not in the ungrouped member table, got:\n%s", name, body)
		}
	}
	if strings.Contains(body, "web") {
		t.Errorf("tagged service web leaked into the ungrouped member table, got:\n%s", body)
	}
}

// A project with services but no profiles shows the ungrouped row, not the
// old "no groups yet" overview - the overview is gone, and a file with
// services always has at least the ungrouped row to select.
func TestServicesWithoutProfilesShowTheUngroupedRowNotTheOverview(t *testing.T) {
	m := panelWith([]types.ServiceConfig{
		{Name: "radarr", Image: "linuxserver/radarr:latest"},
		{Name: "sonarr", Image: "linuxserver/sonarr:latest"},
	}, nil)

	body := ansi.Strip(m.renderBody())

	if strings.Contains(body, "no groups yet") {
		t.Errorf("the service overview must be gone, got:\n%s", body)
	}
	if !strings.Contains(body, "Select a group") {
		t.Errorf("expected the nothing-selected prompt, got:\n%s", body)
	}
}
