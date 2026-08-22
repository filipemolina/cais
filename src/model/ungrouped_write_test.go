package model

import (
	"strings"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/confirmmodal"

	"charm.land/lipgloss/v2"
)

// materializedProject is a loaded project where the reserved ungrouped
// profile has been written onto the profile-less services: the row is backed
// by a real tag rather than derived.
func materializedProject() *types.Project {
	return &types.Project{
		Services: types.Services{
			"web":   types.ServiceConfig{Name: "web", Profiles: []string{"core"}},
			"db":    types.ServiceConfig{Name: "db", Profiles: []string{apptypes.UngroupedGroup}},
			"cache": types.ServiceConfig{Name: "cache", Profiles: []string{apptypes.UngroupedGroup}},
		},
	}
}

// withProjectLoaded is the app on Home with the given project loaded and the
// config sync driven back the way the runtime would.
func withProjectLoaded(t *testing.T, project *types.Project) AppModel {
	t.Helper()

	m := startup(120, 40)
	updated, cmd := m.Update(cmds.GetConfigMsg{FileName: "compose.yaml", Project: project})
	m = drive(updated, collect(cmd)...)

	return applyLayout(m)
}

// A on the derived ungrouped row opens the adopt confirm: the message names
// the services to be tagged and spells out the docker compose up consequence.
func TestToggleUngroupedRequestInDerivedModeOpensAdoptConfirm(t *testing.T) {
	m := withProjectLoaded(t, &types.Project{
		Services: types.Services{
			"web":   types.ServiceConfig{Name: "web", Profiles: []string{"core"}},
			"proxy": types.ServiceConfig{Name: "proxy"},
			"cache": types.ServiceConfig{Name: "cache"},
		},
	})

	_, cmd := m.Update(cmds.ToggleUngroupedRequestMsg{})

	var confirm *cmds.OpenConfirmModalMsg
	for _, msg := range collect(cmd) {
		if c, ok := msg.(cmds.OpenConfirmModalMsg); ok {
			confirm = &c
		}
	}
	if confirm == nil {
		t.Fatal("ToggleUngroupedRequestMsg in derived mode did not open a confirm")
	}
	if !strings.Contains(confirm.Message, "docker compose up") {
		t.Errorf("adopt confirm does not spell out the compose up consequence: %q", confirm.Message)
	}
	if !strings.Contains(confirm.Message, "2 services") {
		t.Errorf("adopt confirm does not name the service count: %q", confirm.Message)
	}
	if confirm.Follow == nil {
		t.Error("adopt confirm has no follow-up command")
	}
}

// A on the materialized ungrouped row opens the release confirm instead.
func TestToggleUngroupedRequestInMaterializedModeOpensReleaseConfirm(t *testing.T) {
	m := withProjectLoaded(t, materializedProject())

	_, cmd := m.Update(cmds.ToggleUngroupedRequestMsg{})

	var confirm *cmds.OpenConfirmModalMsg
	for _, msg := range collect(cmd) {
		if c, ok := msg.(cmds.OpenConfirmModalMsg); ok {
			confirm = &c
		}
	}
	if confirm == nil {
		t.Fatal("ToggleUngroupedRequestMsg in materialized mode did not open a confirm")
	}
	if !strings.Contains(confirm.Message, "Remove the ungrouped profile") {
		t.Errorf("release confirm has the wrong message: %q", confirm.Message)
	}
	if confirm.Follow == nil {
		t.Error("release confirm has no follow-up command")
	}
}

// A successful adopt reloads the config, so the groups list re-derives from
// the updated file.
func TestAdoptUngroupedSuccessReloadsConfig(t *testing.T) {
	m := withProjectLoaded(t, materializedProject())

	_, cmd := m.Update(cmds.AdoptUngroupedMsg{})

	var reloaded bool
	for _, msg := range collect(cmd) {
		if _, ok := msg.(cmds.GetConfigMsg); ok {
			reloaded = true
		}
	}
	if !reloaded {
		t.Error("a successful adopt did not trigger a config reload")
	}
}

// A successful release reloads the config the same way.
func TestReleaseUngroupedSuccessReloadsConfig(t *testing.T) {
	m := withProjectLoaded(t, materializedProject())

	_, cmd := m.Update(cmds.ReleaseUngroupedMsg{})

	var reloaded bool
	for _, msg := range collect(cmd) {
		if _, ok := msg.(cmds.GetConfigMsg); ok {
			reloaded = true
		}
	}
	if !reloaded {
		t.Error("a successful release did not trigger a config reload")
	}
}

// While the ungrouped profile is materialized, a group edit must normalize
// the reserved profile before the reload reads the file - otherwise a service
// that joined another group would keep the ungrouped tag, and one that left
// every group would be left profile-less.
func TestEditGroupSuccessWhileMaterializedNormalizesFirst(t *testing.T) {
	m := withProjectLoaded(t, materializedProject())

	_, cmd := m.Update(cmds.EditGroupMsg{})

	var normalized, reloaded bool
	for _, msg := range collect(cmd) {
		switch msg.(type) {
		case cmds.NormalizeUngroupedMsg:
			normalized = true
		case cmds.GetConfigMsg:
			reloaded = true
		}
	}
	if !normalized {
		t.Error("a materialized edit did not fire the ungrouped normalization")
	}
	if reloaded {
		t.Error("a materialized edit reloaded before normalizing")
	}
}

// While the row is derived there is nothing to normalize: the edit reloads
// directly, and the derived row re-derives from the updated file.
func TestEditGroupSuccessWhileDerivedReloadsDirectly(t *testing.T) {
	m := withProjectLoaded(t, &types.Project{
		Services: types.Services{
			"web":   types.ServiceConfig{Name: "web", Profiles: []string{"core"}},
			"proxy": types.ServiceConfig{Name: "proxy"},
		},
	})

	_, cmd := m.Update(cmds.EditGroupMsg{})

	var normalized, reloaded bool
	for _, msg := range collect(cmd) {
		switch msg.(type) {
		case cmds.NormalizeUngroupedMsg:
			normalized = true
		case cmds.GetConfigMsg:
			reloaded = true
		}
	}
	if normalized {
		t.Error("a derived edit fired the ungrouped normalization")
	}
	if !reloaded {
		t.Error("a derived edit did not reload the config")
	}
}

// A successful normalization reloads the config, completing the
// edit -> normalize -> reload chain.
func TestNormalizeUngroupedSuccessReloadsConfig(t *testing.T) {
	m := withProjectLoaded(t, materializedProject())

	_, cmd := m.Update(cmds.NormalizeUngroupedMsg{})

	var reloaded bool
	for _, msg := range collect(cmd) {
		if _, ok := msg.(cmds.GetConfigMsg); ok {
			reloaded = true
		}
	}
	if !reloaded {
		t.Error("a successful normalization did not trigger a config reload")
	}
}

// An adopt failure surfaces in the error banner and does not reload.
func TestAdoptUngroupedFailureShowsError(t *testing.T) {
	m := withProjectLoaded(t, materializedProject())

	updated, _ := m.Update(cmds.AdoptUngroupedMsg{Err: errBoom{}})
	m = updated.(AppModel)

	if m.lastError == "" {
		t.Error("a failed adopt left no error")
	}
}

// Both ungrouped confirms have to fit an 80-column terminal. confirmmodal
// renders the message it is handed and ModalSurface sets no width, so
// lipgloss pads but never wraps - a long line makes a modal wider than the
// screen it is centred on, and renderWithModal clamps the offset to 0 rather
// than shrinking it ("Narrow terminals: shed whole things" in
// docs/DESIGN.md). The adopt copy is the longest in the app, so it is the one
// that would have gone over.
func TestUngroupedConfirmsFitANarrowTerminal(t *testing.T) {
	cases := []struct {
		name    string
		project *types.Project
	}{
		{"adopt", &types.Project{Services: types.Services{
			"web":   types.ServiceConfig{Name: "web", Profiles: []string{"core"}},
			"proxy": types.ServiceConfig{Name: "proxy"},
		}}},
		{"release", materializedProject()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := withProjectLoaded(t, tc.project)
			_, cmd := m.Update(cmds.ToggleUngroupedRequestMsg{})

			var confirm *cmds.OpenConfirmModalMsg
			for _, msg := range collect(cmd) {
				if c, ok := msg.(cmds.OpenConfirmModalMsg); ok {
					confirm = &c
				}
			}
			if confirm == nil {
				t.Fatal("no confirm was opened")
			}

			// The snapshot is the reassurance that makes this reversible even
			// if the write is not what the user expected; it is part of the
			// safety copy, not decoration.
			if !strings.Contains(confirm.Message, "backed up") {
				t.Errorf("confirm does not mention the backup: %q", confirm.Message)
			}

			if got := lipgloss.Width(confirmmodal.New(confirm.Message, nil).View().Content); got > 80 {
				t.Errorf("confirm modal is %d columns wide, want <= 80:\n%s", got, confirm.Message)
			}
		})
	}
}
