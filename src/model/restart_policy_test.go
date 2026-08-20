package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// TestRigRestartPolicyCycle drives B end to end: Tab to focus the details
// panel, B advances the service's restart: policy to the next value in the
// cycle and writes it straight into the compose file, no modal involved.
func TestRigRestartPolicyCycle(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(restartPolicyFixture), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	t.Chdir(dir)

	r := newRig(t)
	if !r.WaitFor("web", 3*time.Second) {
		t.Fatalf("groups never rendered. Output:\n%s", r.Output())
	}

	r.Send(keyPress('2')) // Services page
	if !r.WaitFor("PROPERTY", 3*time.Second) {
		t.Fatalf("did not switch to the Services page. Output:\n%s", r.Output())
	}
	r.Send(keyPress(tea.KeyTab))
	// The footer's degradation ladder sheds B before h at this terminal
	// width (Boot is the rightmost verb), so healthcheck_test.go's proof of
	// focus - h healthcheck staying lit - doubles as this test's too.
	if !r.WaitFor("h healthcheck", 3*time.Second) {
		t.Fatalf("details panel never took focus (h not advertised). Output:\n%s", r.Output())
	}

	r.Send(letterKey('B'))

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		contents, err := os.ReadFile(filepath.Join(dir, "compose.yaml"))
		if err == nil && strings.Contains(string(contents), "restart: on-failure") {
			return // success: unset -> on-failure, the cycle's first step
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("restart policy was not written to %s. Output:\n%s", dir, r.Output())
}

const restartPolicyFixture = `services:
  web:
    image: nginx:alpine
`
