package model

import (
	"testing"

	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/addservicemodal"
)

// servicesPageWithProject is the app on Services, laid out, with a loaded
// project - the precondition every n-on-Services test needs.
func servicesPageWithProject(t *testing.T) AppModel {
	t.Helper()

	m := startup(120, 40)
	updated, cmd := m.Update(cmds.SetActivePageMsg("Services"))
	m = drive(updated, collect(cmd)...)
	updated, cmd = m.Update(cmds.GetConfigMsg{FileName: "compose.yaml", Project: groupProject()})
	m = drive(updated, collect(cmd)...)

	return applyLayout(m)
}

// n on the Services page opens the add-service modal - the widened gate on
// keys.List.New (D1 in docs/plans/image-search.md).
func TestPressingNOnServicesOpensTheAddServiceModal(t *testing.T) {
	m := servicesPageWithProject(t)

	updated, cmd := m.Update(letterKey('n'))
	m = drive(updated, collect(cmd)...)

	if _, ok := m.activeModal.(addservicemodal.Model); !ok {
		t.Fatalf("activeModal is %T, want addservicemodal.Model", m.activeModal)
	}
}

// n does nothing without a loaded compose file - the same guard
// OpenCreateGroupModalMsg uses on Home, and every other write-path modal in
// the app.
func TestPressingNOnServicesWithNoFileLoadedOpensNothing(t *testing.T) {
	m := startup(120, 40)
	updated, cmd := m.Update(cmds.SetActivePageMsg("Services"))
	m = drive(updated, collect(cmd)...)

	updated, cmd = m.Update(letterKey('n'))
	m = drive(updated, collect(cmd)...)

	if m.activeModal != nil {
		t.Fatalf("activeModal = %T, want nil with no compose file loaded", m.activeModal)
	}
}

// n still creates a group on Home - the widened gate must not have replaced
// that binding's original meaning there.
func TestPressingNOnHomeStillOpensTheCreateGroupModal(t *testing.T) {
	m := withGroupsLoaded(t)

	updated, cmd := m.Update(letterKey('n'))
	m = drive(updated, collect(cmd)...)

	if m.activeModal == nil {
		t.Fatal("n on Home did not open a modal")
	}
	if _, ok := m.activeModal.(addservicemodal.Model); ok {
		t.Fatal("n on Home opened the add-service modal instead of the group modal")
	}
}
