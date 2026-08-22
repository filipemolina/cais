package model

import (
	"testing"

	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/errormodal"
)

// A group action with members resolves them and raises the pending-action
// spinner, exactly as before - the change is only in what the command names.
// The returned model carries the pending action synchronously; the commands
// are not executed here because RunDockerAction shells out to real docker.
func TestGroupActionWithMembersRaisesPendingAction(t *testing.T) {
	m := withGroupsLoaded(t)

	updated, _ := m.Update(cmds.RunDockerActionMsg{Action: "start", Target: "core", IsGroup: true})
	m = updated.(AppModel)

	if m.pendingAction == nil {
		t.Fatal("group action with members did not set pendingAction")
	}
	if m.pendingAction.Target != "core" || m.pendingAction.Action != "start" {
		t.Errorf("pendingAction = %+v, want start/core", m.pendingAction)
	}
}

// A group with no members must not run the command at all. `up -d` with no
// service names is "start the default set" - every service with no profiles:
// key - which is the exact over-reach naming the members removed, only now
// with no group named at all. The refusal is a foreground error about the
// group, not a docker failure, and no spinner is raised or left behind.
func TestEmptyGroupActionIsRefusedRatherThanRun(t *testing.T) {
	m := withGroupsLoaded(t)

	// "ghost" is not a profile in groupProject, so it has no members - the
	// same state as a group whose last service was deleted or edited away.
	updated, _ := m.Update(cmds.RunDockerActionMsg{Action: "start", Target: "ghost", IsGroup: true})
	m = updated.(AppModel)

	if m.pendingAction != nil {
		t.Fatalf("empty group action set pendingAction %+v, want nil", m.pendingAction)
	}

	if _, ok := m.activeModal.(errormodal.Model); !ok {
		t.Fatalf("activeModal is %T, want errormodal.Model with the refusal message", m.activeModal)
	}
}
