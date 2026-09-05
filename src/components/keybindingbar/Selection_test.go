package keybindingbar

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/cais/src/cmds"
)

// hintsOf renders the bar's hints for the current state, the way View does.
func hintsOf(t *testing.T, m Model) string {
	t.Helper()

	var out []string
	for _, binding := range m.bindingsFor() {
		out = append(out, binding.Help().Key+" "+binding.Help().Desc)
	}

	return strings.Join(out, " · ")
}

// driveBar feeds messages through the bar the way AppModel does.
func driveBar(t *testing.T, m Model, msgs ...tea.Msg) Model {
	t.Helper()

	for _, msg := range msgs {
		updated, _ := m.Update(msg)
		next, ok := updated.(Model)
		if !ok {
			t.Fatalf("expected a Model, got %T", updated)
		}
		m = next
	}

	return m
}

// Deleting the last service has to take the action keys off the footer.
//
// configSyncCmds broadcasts the emptied list and the zero selection in one
// batch, and tea.Batch promises no ordering between them - so a handler that
// reads any SetSelectedServiceMsg as a selection leaves the bar advertising
// verbs the details panel will ignore, on roughly half of all deletions.
func TestEmptyingTheServicesListClearsTheActionKeys(t *testing.T) {
	m := New().(Model)

	m = driveBar(t, m,
		cmds.SetActivePageMsg("Services"),
		cmds.SetServicesListMsg([]types.ServiceConfig{{Name: "web"}}),
		cmds.SetSelectedServiceMsg(types.ServiceConfig{Name: "web"}),
	)

	if !strings.Contains(hintsOf(t, m), "start") {
		t.Fatal("precondition: a selected service should put the action keys on the bar")
	}

	m = driveBar(t, m,
		cmds.SetServicesListMsg([]types.ServiceConfig{}),
		cmds.SetSelectedServiceMsg(types.ServiceConfig{}),
	)

	if hints := hintsOf(t, m); strings.Contains(hints, "start") {
		t.Errorf("the action keys survive an emptied services list: %q", hints)
	}
}
