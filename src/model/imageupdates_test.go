package model

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/compose-spec/compose-go/v2/types"

	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/utils"
)

// withCheckSpied swaps the check seam for a spy that records its calls and
// builds nothing, so a model test can assert "the check was dispatched"
// without running docker or a registry. Restored on cleanup; not safe to use
// in parallel with another swap (see checkImageUpdates).
func withCheckSpied(t *testing.T) *[][]types.ServiceConfig {
	t.Helper()

	var called [][]types.ServiceConfig
	original := checkImageUpdates
	t.Cleanup(func() { checkImageUpdates = original })
	checkImageUpdates = func(services []types.ServiceConfig, _ bool) tea.Cmd {
		called = append(called, services)
		return nil
	}

	return &called
}

// imageProject is a project whose services carry images, so the check has
// something to look at.
func imageProject() *types.Project {
	return &types.Project{
		Services: types.Services{
			"web":   types.ServiceConfig{Name: "web", Image: "nginx:alpine"},
			"cache": types.ServiceConfig{Name: "cache", Image: "redis:7-alpine"},
		},
	}
}

// staleUpdate is the map entry an update check would have left behind.
func staleUpdate(remote string) utils.ImageUpdate {
	return utils.ImageUpdate{
		Service:      "web",
		Image:        "nginx:alpine",
		Repo:         "library/nginx",
		LocalDigest:  "sha256:old",
		RemoteDigest: remote,
		State:        utils.ImageStale,
	}
}

// The check rides startup, a file switch, and a restore - the loads that
// change what services exist - and not the routine post-edit reloads (D2).
func TestImageCheckRidesOnlyTheLoadsThatChangeTheServiceSet(t *testing.T) {
	called := withCheckSpied(t)

	m := startup(120, 40)
	if !m.imageCheckOnLoad {
		t.Fatal("the initial model does not arm the check for its first load")
	}

	updated, _ := m.Update(cmds.GetConfigMsg{FileName: "compose.yaml", Project: imageProject()})
	m = updated.(AppModel)

	if len(*called) != 1 {
		t.Fatalf("the startup load dispatched %d checks, want 1", len(*called))
	}
	if m.imageCheckOnLoad {
		t.Error("the check stayed armed after its load")
	}

	// A post-edit reload changes nothing about what services exist - no
	// second check.
	updated, _ = m.Update(cmds.GetConfigMsg{FileName: "compose.yaml", Project: imageProject()})
	m = updated.(AppModel)
	if len(*called) != 1 {
		t.Errorf("the post-edit reload dispatched %d more checks, want 0", len(*called)-1)
	}
}

// A file switch and a restore are the other two loads the check rides (D2).
func TestFileSwitchAndRestoreRecheckImages(t *testing.T) {
	called := withCheckSpied(t)

	m := startup(120, 40)
	updated, _ := m.Update(cmds.GetConfigMsg{FileName: "compose.yaml", Project: imageProject()})
	m = updated.(AppModel)

	// A file switch arms the check for the load it fires.
	updated, _ = m.Update(cmds.SwitchComposeFileMsg{Path: "other.yaml"})
	m = updated.(AppModel)
	if !m.imageCheckOnLoad {
		t.Fatal("a file switch did not arm the check")
	}
	updated, _ = m.Update(cmds.GetConfigMsg{FileName: "other.yaml", Project: imageProject()})
	m = updated.(AppModel)

	// A successful restore arms it too; a failed one does not.
	updated, _ = m.Update(cmds.RestoreBackupMsg{})
	m = updated.(AppModel)
	if !m.imageCheckOnLoad {
		t.Fatal("a successful restore did not arm the check")
	}
	// The armed load consumes the flag, so the failed restore below is
	// judged on its own.
	updated, _ = m.Update(cmds.GetConfigMsg{FileName: "compose.yaml", Project: imageProject()})
	m = updated.(AppModel)

	updated, _ = m.Update(cmds.RestoreBackupMsg{Err: errors.New("nope")})
	m = updated.(AppModel)
	if m.imageCheckOnLoad {
		t.Fatal("a failed restore armed the check")
	}

	updated, _ = m.Update(cmds.GetConfigMsg{FileName: "compose.yaml", Project: imageProject()})
	m = updated.(AppModel)

	// Startup (1) + switch (1) + restore (1) = 3.
	if len(*called) != 3 {
		t.Fatalf("dispatched %d checks, want 3", len(*called))
	}
}

// The five-second container poll is not a load; it must never arm or run
// the check.
func TestThePollDoesNotCheckImages(t *testing.T) {
	called := withCheckSpied(t)

	m := startup(120, 40)
	updated, _ := m.Update(cmds.GetConfigMsg{FileName: "compose.yaml", Project: imageProject()})
	m = updated.(AppModel)

	updated, _ = m.Update(cmds.RefreshContainersTickMsg{})
	m = updated.(AppModel)

	if len(*called) != 1 {
		t.Errorf("a poll tick left %d dispatches behind, want the startup's 1", len(*called))
	}
	if m.imageCheckOnLoad {
		t.Error("a poll tick armed the check")
	}
}

// The check's answer is stored and broadcast to the active page's panels.
func TestImageUpdatesMsgStoresAndBroadcasts(t *testing.T) {
	m := startup(120, 40)

	updates := map[string]utils.ImageUpdate{"web": staleUpdate("sha256:new")}
	updated, cmd := m.Update(cmds.ImageUpdatesMsg{Updates: updates})
	m = updated.(AppModel)

	if m.imageUpdates["web"].State != utils.ImageStale {
		t.Fatalf("the answer was not stored: %+v", m.imageUpdates)
	}

	var broadcast bool
	for _, msg := range collect(cmd) {
		if set, ok := msg.(cmds.SetImageUpdatesMsg); ok && set.Updates["web"].State == utils.ImageStale {
			broadcast = true
		}
	}
	if !broadcast {
		t.Error("the answer did not reach the panels")
	}
}

// A reload drops entries for services that no longer exist; the rest keep
// their answers until the next check replaces them.
func TestConfigReloadDropsEntriesForGoneServices(t *testing.T) {
	m := startup(120, 40)

	m.imageUpdates = map[string]utils.ImageUpdate{
		"web":   staleUpdate("sha256:new"),
		"gone":  staleUpdate("sha256:new"),
		"cache": staleUpdate("sha256:new"),
	}

	updated, _ := m.Update(cmds.GetConfigMsg{FileName: "compose.yaml", Project: imageProject()})
	m = updated.(AppModel)

	if _, ok := m.imageUpdates["gone"]; ok {
		t.Error("the entry for a service the reload removed survived")
	}
	if _, ok := m.imageUpdates["web"]; !ok {
		t.Error("the entry for a service the reload kept was dropped")
	}
}

// A successful update reloads the file the pin changed and clears the glyph
// from the model's own map - the pinned reference now resolves locally to
// exactly the digest it was pinned to, no registry call (D7).
func TestImageUpdatedMsgSuccessClearsTheGlyph(t *testing.T) {
	m := startup(120, 40)
	m.imageUpdates = map[string]utils.ImageUpdate{"web": staleUpdate("sha256:new")}
	m.pendingUpdate = "web"
	m.pendingAction = &chrome.PendingAction{Action: "update", Target: "web"}

	updated, cmd := m.Update(cmds.ImageUpdatedMsg{Service: "web"})
	m = updated.(AppModel)

	if m.pendingUpdate != "" {
		t.Errorf("pendingUpdate = %q after the chain ended", m.pendingUpdate)
	}
	if m.pendingAction != nil {
		t.Error("the spinner outlived the update chain")
	}
	if got := m.imageUpdates["web"]; got.State != utils.ImageUpToDate || got.LocalDigest != "sha256:new" {
		t.Errorf("the glyph did not clear from the pinned digest: %+v", got)
	}

	var broadcast, reload, reprobe bool
	for _, msg := range collect(cmd) {
		switch got := msg.(type) {
		case cmds.SetImageUpdatesMsg:
			broadcast = got.Updates["web"].State == utils.ImageUpToDate
		case cmds.GetConfigMsg:
			reload = true
		case cmds.GetRunningContainersMsg:
			reprobe = true
		}
	}
	if !broadcast {
		t.Error("the cleared map did not reach the panels")
	}
	if !reload {
		t.Error("the pin's write did not reload the config")
	}
	if !reprobe {
		t.Error("the recreated container did not re-probe")
	}
}

// A failed update is foreground: the glyph stays exactly as the check left
// it, and the pin (if the write succeeded, the pull is what failed) stays in
// the file for U or p to retry.
func TestImageUpdatedMsgErrorStaysForegroundAndKeepsTheGlyph(t *testing.T) {
	m := startup(120, 40)
	m.imageUpdates = map[string]utils.ImageUpdate{"web": staleUpdate("sha256:new")}
	m.pendingUpdate = "web"

	updated, _ := m.Update(cmds.ImageUpdatedMsg{Service: "web", Err: errors.New("pull failed")})
	m = updated.(AppModel)

	if m.activeModal == nil {
		t.Error("a failed update left no foreground error")
	}
	if m.pendingUpdate != "" {
		t.Errorf("pendingUpdate = %q after the chain ended", m.pendingUpdate)
	}
	if got := m.imageUpdates["web"]; got.State != utils.ImageStale {
		t.Errorf("a failed update changed the glyph: %+v", got)
	}
}

// Confirm raises the pending-action spinner for the whole chain (it runs
// inside the follow command, off Update's path) and dispatches the follow;
// cancel runs nothing and raises nothing.
func TestUpdateConfirmStagesTheSpinnerAndDispatchesTheFollow(t *testing.T) {
	m := startup(120, 40)
	m.pendingUpdate = "web"

	sentinel := func() tea.Msg { return "follow-ran" }

	updated, cmd := m.Update(cmds.CloseModalMsg{Follow: sentinel})
	m = updated.(AppModel)

	if m.pendingAction == nil || m.pendingAction.Action != "update" {
		t.Fatalf("confirming did not raise the update spinner: %+v", m.pendingAction)
	}

	var followed bool
	for _, msg := range collect(cmd) {
		if s, ok := msg.(string); ok && s == "follow-ran" {
			followed = true
		}
	}
	if !followed {
		t.Error("the confirmed follow command was not dispatched")
	}

	// Cancel: no spinner, no follow, and the staging cleared so a later
	// modal close cannot mistake itself for an update's.
	m = startup(120, 40)
	m.pendingUpdate = "web"
	updated, cmd = m.Update(cmds.CloseModalMsg{})
	m = updated.(AppModel)

	if m.pendingAction != nil {
		t.Error("cancelling raised a spinner")
	}
	if m.pendingUpdate != "" {
		t.Errorf("cancelling left pendingUpdate = %q", m.pendingUpdate)
	}
	for _, msg := range collect(cmd) {
		if s, ok := msg.(string); ok && s == "follow-ran" {
			t.Error("the cancelled follow command ran")
		}
	}
}

// The confirm names both consequences - the file gains a pin, the container
// is recreated (D7) - and the follow it carries is built for the service
// that was selected.
func TestRequestUpdateImageOpensTheConfirm(t *testing.T) {
	m := startup(120, 40)
	m.imageUpdates = map[string]utils.ImageUpdate{"web": staleUpdate("sha256:new")}

	updated, cmd := m.Update(cmds.RequestUpdateImageMsg{Service: "web"})
	m = updated.(AppModel)

	if m.pendingUpdate != "web" {
		t.Errorf("pendingUpdate = %q, want web", m.pendingUpdate)
	}

	// The confirm opens when its message lands, one hop later.
	var message string
	for _, msg := range collect(cmd) {
		if open, ok := msg.(cmds.OpenConfirmModalMsg); ok {
			message = open.Message
			m2, _ := m.Update(open)
			m = m2.(AppModel)
		}
	}
	if m.activeModal == nil {
		t.Fatal("the update request did not open a modal")
	}
	if !strings.Contains(message, "pin") || !strings.Contains(message, "recreated") {
		t.Errorf("the confirm does not state both consequences: %q", message)
	}
}

// The check rides the merged services list the same way the lists do: a
// profiled service is not in Services (02-dependency-guard.md §R2), so its
// update is only ever seen through its DisabledServices entry.
func TestTheCheckSeesDisabledServices(t *testing.T) {
	called := withCheckSpied(t)

	m := startup(120, 40)

	project := &types.Project{
		Services:         types.Services{"web": types.ServiceConfig{Name: "web", Image: "nginx:alpine"}},
		DisabledServices: types.Services{"db": types.ServiceConfig{Name: "db", Image: "postgres:16"}},
	}
	updated, _ := m.Update(cmds.GetConfigMsg{FileName: "compose.yaml", Project: project})
	m = updated.(AppModel)

	var sawWeb, sawDB bool
	for _, svc := range (*called)[0] {
		switch svc.Name {
		case "web":
			sawWeb = true
		case "db":
			sawDB = true
		}
	}
	if !sawWeb || !sawDB {
		t.Errorf("the check saw %d services, want both", len(*called))
	}
}

// The bar offers U only when the selected service's check came back Stale -
// every other state, or a check that never answered, leaves it off. The bar
// does not advertise inert keys (D7). The 160-column terminal is wide enough
// that the page's other verbs are not being shed: the assertion is about
// advertising, not about the bar's degradation order.
func TestTheUpdateKeyAdvertisesOnlyForAStaleSelection(t *testing.T) {
	m := startup(160, 40)
	updated, cmd := m.Update(cmds.SetActivePageMsg("Services"))
	m = drive(updated, collect(cmd)...)
	updated, cmd = m.Update(cmds.GetConfigMsg{FileName: "compose.yaml", Project: groupProject()})
	m = applyLayout(drive(updated, collect(cmd)...))

	name := m.selection.serviceName
	if name == "" {
		t.Fatal("precondition: no service selected on the Services page")
	}

	updated, _ = m.Update(cmds.ImageUpdatesMsg{Updates: map[string]utils.ImageUpdate{
		name: {Service: name, State: utils.ImageStale},
	}})
	m = updated.(AppModel)

	if !m.keyContext().UpdateAvailable {
		t.Error("a stale selection did not arm the update context")
	}
	if footer := ansi.Strip(m.components.KeybindingBar.View().Content); !strings.Contains(footer, "update image") {
		t.Errorf("the bar does not offer U for a stale selection: %q", footer)
	}

	updated, _ = m.Update(cmds.ImageUpdatesMsg{Updates: map[string]utils.ImageUpdate{
		name: {Service: name, State: utils.ImageUnknown},
	}})
	m = updated.(AppModel)

	if m.keyContext().UpdateAvailable {
		t.Error("an Unknown answer armed the update context")
	}
	if footer := ansi.Strip(m.components.KeybindingBar.View().Content); strings.Contains(footer, "update image") {
		t.Errorf("the bar offers U with no stale answer: %q", footer)
	}
}

// U yields to whoever owns the keyboard: a filter being typed turns it into a
// letter, and a modal gets it exclusively - the confirm on screen is the only
// thing U can reach (D7's routing).
func TestUYieldsToAFilterAndToAModal(t *testing.T) {
	m := servicesPageWithProject(t)
	name := m.selection.serviceName
	m.imageUpdates = map[string]utils.ImageUpdate{name: {Service: name, State: utils.ImageStale}}

	m = drive(m, letter('/'))
	if !m.keyboardOwned() {
		t.Fatal("precondition: / did not hand the keyboard to the list")
	}

	updated, cmd := m.Update(letterKey('U'))
	m = updated.(AppModel)
	for _, msg := range collect(cmd) {
		switch msg.(type) {
		case cmds.RequestUpdateImageMsg, cmds.OpenConfirmModalMsg:
			t.Error("U reached the panel while a filter was being typed")
		}
	}

	// With the update confirm open, U reaches the modal only: no second
	// request, and the confirm on screen is still the same one.
	updated, cmd = m.Update(cmds.RequestUpdateImageMsg{Service: name})
	m = updated.(AppModel)
	for _, msg := range collect(cmd) {
		if open, ok := msg.(cmds.OpenConfirmModalMsg); ok {
			next, _ := m.Update(open)
			m = next.(AppModel)
		}
	}
	if m.activeModal == nil {
		t.Fatal("the confirm did not open")
	}

	updated, cmd = m.Update(letterKey('U'))
	m = updated.(AppModel)
	for _, msg := range collect(cmd) {
		switch msg.(type) {
		case cmds.RequestUpdateImageMsg, cmds.OpenConfirmModalMsg:
			t.Error("U reached the panel while a modal was open")
		}
	}
	// The confirm is still what holds the screen - U could neither dismiss
	// it (a modal owns the keyboard) nor open anything over it.
	if m.activeModal == nil {
		t.Error("U while the confirm was open closed it")
	}
}
