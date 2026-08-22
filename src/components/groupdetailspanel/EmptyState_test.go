package groupdetailspanel

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/cais/src/apptypes"
)

// panelWith builds a Model in the state renderBody needs: laid out, with the
// given services and containers. No group is selected, so renderBody's
// nothing-selected branch is what the tests below exercise.
func panelWith(services []types.ServiceConfig, containers []apptypes.DockerContainer) Model {
	m := New().(Model)
	m.panelWidth = 60
	m.panelHeight = 24
	m.services = services
	m.containers = containers
	return m
}

// A compose file with no services at all - a fresh or newly bootstrapped
// file - has nothing to list, so the panel falls back to the original
// onboarding card explaining what a group is.
func TestNoServicesShowsGettingStartedCard(t *testing.T) {
	m := panelWith(nil, nil)

	body := ansi.Strip(m.renderBody())

	if !strings.Contains(body, "Getting started") {
		t.Errorf("expected the onboarding card with no services, got:\n%s", body)
	}
}

// Once any service carries a profiles: tag, knownGroups() is no longer
// empty and the ordinary "select a group" / member-table path takes over.
func TestSelectAGroupOnceAGroupExists(t *testing.T) {
	m := panelWith([]types.ServiceConfig{
		{Name: "grouped", Image: "img", Profiles: []string{"media"}},
	}, nil)

	body := ansi.Strip(m.renderBody())

	if !strings.Contains(body, "Select a group") {
		t.Errorf("expected the ordinary nothing-selected state, got:\n%s", body)
	}
}
