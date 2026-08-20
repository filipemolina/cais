package detailspanel

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/cmds"
)

// A successful healthcheck insert on a running service sets the apply
// hint - restart (r) reuses the container's existing config, only start
// (s) re-reads compose, and a user who saves a healthcheck and presses r
// would otherwise see nothing happen (docs/plans/healthcheck-insertion.md,
// "The apply gap").
func TestAddHealthcheckMsgSetsTheApplyHintForARunningService(t *testing.T) {
	svc := types.ServiceConfig{Name: "db", Image: "postgres:16"}
	m := Model{
		service:     &svc,
		panelWidth:  100,
		panelHeight: 30,
		containers:  []apptypes.DockerContainer{{Service: "db", State: "running"}},
	}

	updated, _ := m.Update(cmds.AddHealthcheckMsg{ServiceName: "db"})
	got := updated.(Model)

	if got.applyHint == "" {
		t.Fatal("applyHint is empty, want the apply-gap hint")
	}
	if !strings.Contains(got.applyHint, "press s") {
		t.Errorf("applyHint = %q, want it to name s specifically", got.applyHint)
	}

	frame := ansi.Strip(got.View().Content)
	if !strings.Contains(frame, "press s") {
		t.Errorf("apply hint not rendered in the panel footer:\n%s", frame)
	}
}

// A stopped service gets no hint - there is nothing running to apply to
// yet, and the config table already shows the Healthcheck row once the
// reload lands.
func TestAddHealthcheckMsgNoHintForAStoppedService(t *testing.T) {
	svc := types.ServiceConfig{Name: "db", Image: "postgres:16"}
	m := Model{service: &svc, panelWidth: 100, panelHeight: 30}

	updated, _ := m.Update(cmds.AddHealthcheckMsg{ServiceName: "db"})
	got := updated.(Model)

	if got.applyHint != "" {
		t.Errorf("applyHint = %q, want empty for a stopped service", got.applyHint)
	}
}

// An error does not set the hint - nothing was applied.
func TestAddHealthcheckMsgErrorSetsNoHint(t *testing.T) {
	svc := types.ServiceConfig{Name: "db", Image: "postgres:16"}
	m := Model{
		service:     &svc,
		panelWidth:  100,
		panelHeight: 30,
		containers:  []apptypes.DockerContainer{{Service: "db", State: "running"}},
	}

	updated, _ := m.Update(cmds.AddHealthcheckMsg{ServiceName: "db", Err: errBoom{}})
	got := updated.(Model)

	if got.applyHint != "" {
		t.Errorf("applyHint = %q, want empty when the write failed", got.applyHint)
	}
}

// A successful restart-policy cycle on a running service sets the same
// apply hint the healthcheck insert does - restart: is a container-level
// attribute, applied only when the container is recreated.
func TestCycleRestartPolicyMsgSetsTheApplyHintForARunningService(t *testing.T) {
	svc := types.ServiceConfig{Name: "db", Image: "postgres:16"}
	m := Model{
		service:     &svc,
		panelWidth:  100,
		panelHeight: 30,
		containers:  []apptypes.DockerContainer{{Service: "db", State: "running"}},
	}

	updated, _ := m.Update(cmds.CycleRestartPolicyMsg{ServiceName: "db", Policy: "unless-stopped"})
	got := updated.(Model)

	if got.applyHint == "" {
		t.Fatal("applyHint is empty, want the apply-gap hint")
	}
	if !strings.Contains(got.applyHint, "press s") {
		t.Errorf("applyHint = %q, want it to name s specifically", got.applyHint)
	}
}

// A stopped service gets no hint - the config table already shows the new
// policy once the reload lands.
func TestCycleRestartPolicyMsgNoHintForAStoppedService(t *testing.T) {
	svc := types.ServiceConfig{Name: "db", Image: "postgres:16"}
	m := Model{service: &svc, panelWidth: 100, panelHeight: 30}

	updated, _ := m.Update(cmds.CycleRestartPolicyMsg{ServiceName: "db", Policy: "unless-stopped"})
	got := updated.(Model)

	if got.applyHint != "" {
		t.Errorf("applyHint = %q, want empty for a stopped service", got.applyHint)
	}
}

// An error does not set the hint - nothing was applied.
func TestCycleRestartPolicyMsgErrorSetsNoHint(t *testing.T) {
	svc := types.ServiceConfig{Name: "db", Image: "postgres:16"}
	m := Model{
		service:     &svc,
		panelWidth:  100,
		panelHeight: 30,
		containers:  []apptypes.DockerContainer{{Service: "db", State: "running"}},
	}

	updated, _ := m.Update(cmds.CycleRestartPolicyMsg{ServiceName: "db", Err: errBoom{}})
	got := updated.(Model)

	if got.applyHint != "" {
		t.Errorf("applyHint = %q, want empty when the write failed", got.applyHint)
	}
}

// A docker action request clears a stale apply hint - the user is already
// doing something about it.
func TestDockerActionClearsTheApplyHint(t *testing.T) {
	svc := types.ServiceConfig{Name: "db", Image: "postgres:16"}
	m := Model{
		service:     &svc,
		isFocused:   true,
		panelWidth:  100,
		panelHeight: 30,
		containers:  []apptypes.DockerContainer{{Service: "db", State: "running"}},
		applyHint:   "running: press s to apply",
	}

	updated, _ := m.Update(keyPress('s'))
	got := updated.(Model)

	if got.applyHint != "" {
		t.Errorf("applyHint = %q, want cleared after requesting a docker action", got.applyHint)
	}
}
