package model

import (
	"errors"
	"testing"

	"stack-stitcher/src/cmds"
)

// The command returned for OpenEditorMsg is a tea.ExecProcess that would
// launch a real editor, so these tests assert on the model's state and on
// the messages that don't hand over the terminal - never by running it.

func TestOpeningTheEditorSuspendsBackgroundWork(t *testing.T) {
	m := GetInitialModel()
	m.config.configFileName = "compose.yaml"

	m = updateForTest(t, m, cmds.OpenEditorMsg{})

	if !m.externalEditorOpen {
		t.Error("external editor should be marked open")
	}
	if m.shouldPollContainers() {
		t.Error("should not poll while the editor holds the terminal")
	}
}

// Nothing to hand the editor, and launching one on an empty path would open
// an unnamed buffer that saves nowhere useful.
func TestOpeningTheEditorWithoutAComposeFileReportsInstead(t *testing.T) {
	m := GetInitialModel()

	m = updateForTest(t, m, cmds.OpenEditorMsg{})

	if m.externalEditorOpen {
		t.Error("external editor should not be marked open with no compose file")
	}
	if m.lastError == "" {
		t.Error("expected an error explaining there is no compose file")
	}
}

func TestClosingTheEditorReloadsTheConfig(t *testing.T) {
	m := GetInitialModel()
	m.externalEditorOpen = true

	updated, cmd := m.Update(cmds.EditorClosedMsg{})
	m = updated.(AppModel)

	if m.externalEditorOpen {
		t.Error("external editor should be marked closed")
	}

	// GetConfig reads the compose file in the working directory. There
	// isn't one here, so it comes back as an error - which still proves the
	// reload was queued, without needing a fixture on disk.
	var reloaded bool
	for _, msg := range collect(cmd) {
		if _, ok := msg.(cmds.GetConfigMsg); ok {
			reloaded = true
		}
	}

	if !reloaded {
		t.Error("closing the editor did not queue a config reload")
	}
}

func TestAFailedEditorSurfacesItsError(t *testing.T) {
	m := GetInitialModel()
	m.externalEditorOpen = true

	m = updateForTest(t, m, cmds.EditorClosedMsg{Err: errors.New("exec: \"nope\": executable file not found in $PATH")})

	if m.externalEditorOpen {
		t.Error("external editor should be marked closed even when it failed")
	}
	if m.lastError == "" {
		t.Error("a failed editor should reach the banner")
	}
}

// The banner belongs to the poll only when the poll put it there. An editor
// error must not be cleared by the next successful background refresh.
func TestAFailedEditorErrorIsNotOwnedByThePoll(t *testing.T) {
	m := GetInitialModel()
	m = updateForTest(t, m, cmds.EditorClosedMsg{Err: errors.New("editor exploded")})

	m = updateForTest(t, m, cmds.GetRunningContainersMsg{Background: true})

	if m.lastError == "" {
		t.Error("a background poll cleared an editor error it did not cause")
	}
}
