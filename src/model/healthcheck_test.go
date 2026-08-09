package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// TestRigHealthcheckInsertion drives the whole flow end to end
// (docs/plans/healthcheck-insertion.md): Tab to focus the details panel, h
// opens the picker, Enter on the first (image-matched) template writes the
// healthcheck into the compose file and closes the modal.
func TestRigHealthcheckInsertion(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(healthcheckFixture), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	t.Chdir(dir)

	r := newRig(t)
	if !r.WaitFor("db", 3*time.Second) {
		t.Fatalf("groups never rendered. Output:\n%s", r.Output())
	}

	r.Send(keyPress('2')) // Services page
	// "PROPERTY" is the config table header the details panel renders on the
	// Services page only; the tab bar label "Services" is on screen on every
	// page, so waiting on it would race the page switch.
	if !r.WaitFor("PROPERTY", 3*time.Second) {
		t.Fatalf("did not switch to the Services page. Output:\n%s", r.Output())
	}
	r.Send(keyPress(tea.KeyTab)) // focus the details panel
	if !r.WaitFor("h healthcheck", 3*time.Second) {
		t.Fatalf("details panel never took focus (h not advertised). Output:\n%s", r.Output())
	}

	r.Send(letterKey('h'))
	// Both substrings render in the same frame; one WaitFor covers both.
	if !r.WaitFor("Add healthcheck", 3*time.Second) {
		t.Fatalf("healthcheck picker did not open. Output:\n%s", r.Output())
	}
	if !strings.Contains(r.Output(), "PostgreSQL") {
		t.Fatalf("PostgreSQL template not offered for a postgres image. Output:\n%s", r.Output())
	}

	r.Send(keyPress(tea.KeyEnter))
	if !r.WaitForNot("Add healthcheck", 3*time.Second) {
		t.Fatalf("healthcheck picker did not close. Output:\n%s", r.Output())
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		contents, err := os.ReadFile(filepath.Join(dir, "compose.yaml"))
		if err == nil && strings.Contains(string(contents), "pg_isready") {
			return // success
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("healthcheck was not written to %s. Output:\n%s", dir, r.Output())
}

const healthcheckFixture = `services:
  db:
    image: postgres:16
`
