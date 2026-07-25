package model

import (
	"testing"

	"stack-stitcher/src/cmds"

	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

// projectWith builds a loaded project from service name -> group tags.
func projectWith(services map[string][]string) *types.Project {
	project := &types.Project{Services: types.Services{}}

	for name, profiles := range services {
		project.Services[name] = types.ServiceConfig{Name: name, Profiles: profiles}
	}

	return project
}

// syncMsgs runs configSyncCmds and flattens the result, the way the runtime
// would before delivering it to the panels.
func syncMsgs(m AppModel) []tea.Msg {
	var msgs []tea.Msg
	for _, cmd := range m.configSyncCmds() {
		msgs = append(msgs, collect(cmd)...)
	}

	return msgs
}

func selectedServiceFrom(msgs []tea.Msg) (string, bool) {
	for _, msg := range msgs {
		if service, ok := msg.(cmds.SetSelectedServiceMsg); ok {
			return types.ServiceConfig(service).Name, true
		}
	}

	return "", false
}

func selectedGroupFrom(msgs []tea.Msg) (string, bool) {
	for _, msg := range msgs {
		if group, ok := msg.(cmds.SetSelectedGroupMsg); ok {
			return string(group), true
		}
	}

	return "", false
}

// Every write to the compose file reloads the config. Without this, the
// reload re-selects the alphabetically first service, so the user is thrown
// back to the top of the list the moment they change anything - and never
// sees the change they just made to the service they were looking at.
func TestReloadKeepsTheSelectedService(t *testing.T) {
	m := AppModel{selection: selectionModel{serviceName: "web"}}
	m.config.configProject = projectWith(map[string][]string{
		"api": nil, "db": nil, "web": nil,
	})

	got, ok := selectedServiceFrom(syncMsgs(m))
	if !ok {
		t.Fatal("reload sent no service selection")
	}
	if want := "web"; got != want {
		t.Errorf("selected service after reload: got %q, want %q", got, want)
	}
}

func TestReloadKeepsTheSelectedGroup(t *testing.T) {
	m := AppModel{selection: selectionModel{groupName: "media"}}
	m.config.configProject = projectWith(map[string][]string{
		"api": {"core"}, "plex": {"media"},
	})

	got, ok := selectedGroupFrom(syncMsgs(m))
	if !ok {
		t.Fatal("reload sent no group selection")
	}
	if want := "media"; got != want {
		t.Errorf("selected group after reload: got %q, want %q", got, want)
	}
}

// The selection can vanish - the service was removed or renamed outside the
// app - and the first entry is the only answer left.
func TestReloadFallsBackWhenTheSelectionIsGone(t *testing.T) {
	m := AppModel{selection: selectionModel{serviceName: "web", groupName: "media"}}
	m.config.configProject = projectWith(map[string][]string{
		"api": {"core"}, "db": {"core"},
	})

	msgs := syncMsgs(m)

	if got, _ := selectedServiceFrom(msgs); got != "api" {
		t.Errorf("selected service after the old one vanished: got %q, want %q", got, "api")
	}
	if got, _ := selectedGroupFrom(msgs); got != "core" {
		t.Errorf("selected group after the old one vanished: got %q, want %q", got, "core")
	}
}

// Nothing selected yet, e.g. the first load.
func TestReloadSelectsTheFirstEntryWithNoPriorSelection(t *testing.T) {
	m := AppModel{}
	m.config.configProject = projectWith(map[string][]string{
		"api": {"core"}, "db": {"core"},
	})

	msgs := syncMsgs(m)

	if got, _ := selectedServiceFrom(msgs); got != "api" {
		t.Errorf("selected service on first load: got %q, want %q", got, "api")
	}
	if got, _ := selectedGroupFrom(msgs); got != "core" {
		t.Errorf("selected group on first load: got %q, want %q", got, "core")
	}
}

func TestReloadOfAnEmptyProjectSelectsNothing(t *testing.T) {
	m := AppModel{selection: selectionModel{serviceName: "web", groupName: "media"}}
	m.config.configProject = projectWith(nil)

	msgs := syncMsgs(m)

	if got, ok := selectedServiceFrom(msgs); ok {
		t.Errorf("empty project selected service %q, want no selection message", got)
	}
	if got, ok := selectedGroupFrom(msgs); ok {
		t.Errorf("empty project selected group %q, want no selection message", got)
	}
}

// AppModel does not originate these messages - it watches the ones the
// panels emit on their way past, which is what gives the reload something
// to restore.
func TestAppModelRecordsWhatThePanelsSelect(t *testing.T) {
	m := drive(GetInitialModel(),
		cmds.SetSelectedServiceMsg(types.ServiceConfig{Name: "web"}),
		cmds.SetSelectedGroupMsg("media"),
	)

	if got, want := m.selection.serviceName, "web"; got != want {
		t.Errorf("recorded service: got %q, want %q", got, want)
	}
	if got, want := m.selection.groupName, "media"; got != want {
		t.Errorf("recorded group: got %q, want %q", got, want)
	}
}
