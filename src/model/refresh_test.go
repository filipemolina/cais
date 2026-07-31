package model

import (
	"errors"
	"testing"

	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components/confirmmodal"
	"github.com/filipemolina/stack-stitcher/src/utils"

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
	m := GetInitialModel(utils.ComposeSource{})

	if m.shouldPollContainers() {
		t.Error("should not poll before a compose project has loaded")
	}

	m.config.configProject = &types.Project{}
	if !m.shouldPollContainers() {
		t.Error("should poll after a compose project has loaded")
	}

	m.activeModal = confirmmodal.New("Confirm?", nil)
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
	m := GetInitialModel(utils.ComposeSource{})

	m = updateForTest(t, m, cmds.GetRunningContainersMsg{
		Err:        errors.New("docker daemon unavailable"),
		Background: true,
	})

	// The poll error goes to the banner.
	if m.lastError != "docker daemon unavailable" {
		t.Fatalf("poll error = %q, want docker daemon unavailable", m.lastError)
	}

	// A foreground docker action error opens a modal, not the banner.
	m = updateForTest(t, m, cmds.DockerActionMsg{Err: errors.New("docker start failed")})

	if m.activeModal == nil {
		t.Fatal("action error should open an error modal")
	}

	// The banner should still have the poll error (the modal did not touch it).
	if m.lastError != "docker daemon unavailable" {
		t.Errorf("banner after action error = %q, want poll error preserved", m.lastError)
	}

	// A successful background poll clears its own error from the banner.
	m = updateForTest(t, m, cmds.GetRunningContainersMsg{
		Containers: []apptypes.DockerContainer{},
		Background: true,
	})

	if m.lastError != "" {
		t.Errorf("background success did not clear poll error: %q", m.lastError)
	}
}

func TestBackgroundPollClearsItsOwnRecoveredError(t *testing.T) {
	m := GetInitialModel(utils.ComposeSource{})

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
	m := GetInitialModel(utils.ComposeSource{})
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
