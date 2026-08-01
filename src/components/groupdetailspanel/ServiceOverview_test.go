package groupdetailspanel

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/stack-stitcher/src/apptypes"
)

// panelWith builds a Model in the state renderBody needs: laid out, with the
// given services and containers. No group is selected and none of these
// services carry a profiles: tag, so renderBody's knownGroups() == 0 branch
// is what every test below exercises.
func panelWith(services []types.ServiceConfig, containers []apptypes.DockerContainer) Model {
	m := New().(Model)
	m.panelWidth = 60
	m.panelHeight = 24
	m.services = services
	m.containers = containers
	return m
}

// When the compose file has services but no groups reference any of them,
// the overview replaces the plain onboarding card with the service count and
// a table of what's there.
func TestServiceOverviewShowsCountAndServices(t *testing.T) {
	m := panelWith([]types.ServiceConfig{
		{Name: "radarr", Image: "linuxserver/radarr:latest"},
		{Name: "sonarr", Image: "linuxserver/sonarr:latest"},
		{Name: "plex", Image: "plexinc/pms-docker:latest"},
	}, nil)

	body := ansi.Strip(m.renderBody())

	if !strings.Contains(body, "3 services, no groups yet") {
		t.Errorf("missing the service count header, got:\n%s", body)
	}
	for _, name := range []string{"radarr", "sonarr", "plex"} {
		if !strings.Contains(body, name) {
			t.Errorf("service %q not listed, got:\n%s", name, body)
		}
	}
}

// Singular count reads naturally.
func TestServiceOverviewSingularCount(t *testing.T) {
	m := panelWith([]types.ServiceConfig{{Name: "web", Image: "nginx:alpine"}}, nil)

	body := ansi.Strip(m.renderBody())

	if !strings.Contains(body, "1 service, no groups yet") {
		t.Errorf("expected the singular form, got:\n%s", body)
	}
}

// Running state comes from the live containers, same as the populated member
// table - a service the daemon reports running shows the running dot color
// and state text, one not reported shows stopped.
func TestServiceOverviewShowsRunningState(t *testing.T) {
	m := panelWith([]types.ServiceConfig{
		{Name: "web", Image: "nginx:alpine"},
		{Name: "db", Image: "postgres:16"},
	}, []apptypes.DockerContainer{
		{Service: "web", State: "running"},
	})

	body := ansi.Strip(m.renderBody())

	if !strings.Contains(body, "running") {
		t.Errorf("expected web's running state in the overview, got:\n%s", body)
	}
	if !strings.Contains(body, "stopped") {
		t.Errorf("expected db's stopped state in the overview, got:\n%s", body)
	}
}

// n is still the only key advertised, and it still means "new group" here -
// the overview is read-only, so the hint is the whole action surface.
func TestServiceOverviewAdvertisesNAndPageSwitch(t *testing.T) {
	m := panelWith([]types.ServiceConfig{{Name: "web", Image: "nginx:alpine"}}, nil)

	body := ansi.Strip(m.renderBody())

	if !strings.Contains(body, "create your first group") {
		t.Errorf("missing the n hint, got:\n%s", body)
	}
	if !strings.Contains(body, "browse services") {
		t.Errorf("missing the 2 hint, got:\n%s", body)
	}
}

// A compose file with no services at all - a fresh or newly bootstrapped
// file - has nothing to overview, so it falls back to the original
// onboarding card explaining what a group is.
func TestServiceOverviewFallsBackWhenThereAreNoServices(t *testing.T) {
	m := panelWith(nil, nil)

	body := ansi.Strip(m.renderBody())

	if !strings.Contains(body, "Getting started") {
		t.Errorf("expected the onboarding fallback with no services, got:\n%s", body)
	}
	if strings.Contains(body, "no groups yet") {
		t.Errorf("should not show the overview header with no services, got:\n%s", body)
	}
}

// Once any service carries a profiles: tag, knownGroups() is no longer
// empty and the ordinary "select a group" / member-table path takes over -
// the overview must not show through it.
func TestServiceOverviewDoesNotShowOnceAGroupExists(t *testing.T) {
	m := panelWith([]types.ServiceConfig{
		{Name: "grouped", Image: "img", Profiles: []string{"media"}},
	}, nil)

	body := ansi.Strip(m.renderBody())

	if strings.Contains(body, "no groups yet") {
		t.Errorf("overview should not show once a group exists, got:\n%s", body)
	}
	if !strings.Contains(body, "Select a group") {
		t.Errorf("expected the ordinary nothing-selected state, got:\n%s", body)
	}
}
