package model

import (
	"errors"
	"testing"

	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components"

	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

func updateForTest(t *testing.T, m AppModel, msg tea.Msg) AppModel {
	t.Helper()

	updated, _ := m.Update(msg)
	result, ok := updated.(AppModel)
	if !ok {
		t.Fatalf("Update returned %T, want model.AppModel", updated)
	}

	return result
}

func TestShouldPollContainers(t *testing.T) {
	m := GetInitialModel()

	if m.shouldPollContainers() {
		t.Error("should not poll before a compose project has loaded")
	}

	m.config.configProject = &types.Project{}
	if !m.shouldPollContainers() {
		t.Error("should poll after a compose project has loaded")
	}

	m.activeModal = components.ConfirmModal("Confirm?", nil)
	if m.shouldPollContainers() {
		t.Error("should not poll while a modal is open")
	}

	m.activeModal = nil
	m.externalEditorOpen = true
	if m.shouldPollContainers() {
		t.Error("should not poll while an external editor holds the terminal")
	}
}

func TestBackgroundPollPreservesActionErrorThatReplacedPollError(t *testing.T) {
	m := GetInitialModel()

	m = updateForTest(t, m, cmds.GetRunningContainersMsg{
		Err:        errors.New("docker daemon unavailable"),
		Background: true,
	})
	m = updateForTest(t, m, cmds.DockerActionMsg{Err: errors.New("docker start failed")})

	if m.lastError != "docker start failed" {
		t.Fatalf("action error = %q, want docker start failed", m.lastError)
	}
	if m.lastErrorFromPoll {
		t.Fatal("action error was still marked as a poll error")
	}

	m = updateForTest(t, m, cmds.GetRunningContainersMsg{
		Containers: []apptypes.DockerContainer{},
		Background: true,
	})

	if m.lastError != "docker start failed" {
		t.Errorf("background success cleared an unrelated error: %q", m.lastError)
	}
}

func TestBackgroundPollClearsItsOwnRecoveredError(t *testing.T) {
	m := GetInitialModel()

	m = updateForTest(t, m, cmds.GetRunningContainersMsg{
		Err:        errors.New("docker daemon unavailable"),
		Background: true,
	})

	if m.lastError != "docker daemon unavailable" {
		t.Fatalf("poll error = %q, want docker daemon unavailable", m.lastError)
	}
	if !m.lastErrorFromPoll {
		t.Fatal("poll error was not marked as coming from the background poll")
	}

	m = updateForTest(t, m, cmds.GetRunningContainersMsg{
		Containers: []apptypes.DockerContainer{},
		Background: true,
	})

	if m.lastError != "" {
		t.Errorf("recovered background poll left an error banner: %q", m.lastError)
	}
	if m.lastErrorFromPoll {
		t.Error("recovered background poll did not clear its error source")
	}
}

func TestForegroundRefreshClearsAnyExistingError(t *testing.T) {
	m := GetInitialModel()
	m.lastError = "docker daemon unavailable"
	m.lastErrorFromPoll = true

	m = updateForTest(t, m, cmds.GetRunningContainersMsg{
		Containers: []apptypes.DockerContainer{},
	})

	if m.lastError != "" {
		t.Errorf("foreground success left an error banner: %q", m.lastError)
	}
	if m.lastErrorFromPoll {
		t.Error("foreground success did not clear the error source")
	}
}
