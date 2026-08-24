package detailspanel

import (
	"testing"

	"github.com/filipemolina/cais/src/cmds"

	"github.com/compose-spec/compose-go/v2/types"
)

// B does not know the service's current restart: policy or which compose
// file is loaded - both live on AppModel - so it only asks, the same split
// DockerAction_test.go pins for start/stop/restart/pull/remove.
func TestBootKeyRequestsARestartPolicyCycle(t *testing.T) {
	panel := New(&types.ServiceConfig{Name: "web"}, "10.0.0.5")

	_, cmd := panel.Update(keyPress('B'))

	if cmd == nil {
		t.Fatal("expected a command, got nil")
	}
	msg, ok := cmd().(cmds.CycleRestartPolicyRequestMsg)
	if !ok {
		t.Fatalf("expected a CycleRestartPolicyRequestMsg, got %T", cmd())
	}
	if msg.ServiceName != "web" {
		t.Errorf("ServiceName = %q, want %q", msg.ServiceName, "web")
	}
}

// B clears a stale apply hint the same way the docker action keys do - the
// user is already doing something about it.
func TestBootKeyClearsTheApplyHint(t *testing.T) {
	svc := types.ServiceConfig{Name: "web"}
	panel := Model{
		service:     &svc,
		panelWidth:  100,
		panelHeight: 30,
		applyHint:   "running: press s to apply",
	}

	updated, _ := panel.Update(keyPress('B'))

	if got := updated.(Model).applyHint; got != "" {
		t.Errorf("applyHint = %q, want cleared after pressing B", got)
	}
}
