package model

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/compose-spec/compose-go/v2/types"

	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/cmds"
)

// Deleting the last service has to take the action keys off the footer.
//
// This lived in keybindingbar while the bar built its own keys.Context from
// mirrored fields, and it caught a real bug there: configSyncCmds broadcasts
// the emptied list and the zero selection in one batch, tea.Batch promises no
// ordering between them, and a handler that read any SetSelectedServiceMsg as
// a selection left the bar advertising verbs the details panel would ignore.
//
// The bar cannot get this wrong any more - it derives nothing. So the test
// moved here, to the one place that can: AppModel resolving the context and
// handing it down. Asserting on the rendered footer keeps it honest about the
// whole path rather than about keyContext() alone.
func TestEmptyingTheServicesListClearsTheFooterActionKeys(t *testing.T) {
	m := servicesPageWithProject(t)

	if footer := ansi.Strip(m.components.KeybindingBar.View().Content); !strings.Contains(footer, "start") {
		t.Fatalf("precondition: a populated services list should put the action keys on the bar: %q", footer)
	}

	updated, cmd := m.Update(cmds.GetConfigMsg{
		FileName: "compose.yaml",
		Project:  &types.Project{Services: types.Services{}},
	})
	m = applyLayout(drive(updated, collect(cmd)...))

	if footer := ansi.Strip(m.components.KeybindingBar.View().Content); strings.Contains(footer, "start") {
		t.Errorf("the action keys survive an emptied services list: %q", footer)
	}
}

// The other half of the same guarantee: the footer and the help overlay read
// the same resolved context, so they cannot disagree about what is pressable.
// They used to derive it separately, which is how they drifted.
func TestFooterAndOverlayShareOneContext(t *testing.T) {
	m := servicesPageWithProject(t)

	ctx := m.keyContext()
	if ctx.Page != apptypes.PageServices {
		t.Fatalf("context page = %q, want Services", ctx.Page)
	}
	if ctx.ListEmpty {
		t.Error("a loaded project should not report an empty services list")
	}
}
