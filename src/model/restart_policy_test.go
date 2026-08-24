package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	// Both body panels are always active, so no focus step is needed; the
	// selected service's verbs - including B boot - are advertised immediately.
	if !r.WaitFor("B boot", 3*time.Second) {
		t.Fatalf("boot verb not advertised for the selected service. Output:\n%s", r.Output())
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
