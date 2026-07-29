package model

import (
	"errors"
	"testing"

	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/utils"
)

// A background poll suppresses GetRunningContainersMsg so components wait for
// the enriched GetContainerStatsMsg instead. When the stats fetch fails, that
// enriched message never carries any containers - so the poll's data must not
// be dropped on the floor. The running count is the observable proof: it is
// derived from the poll, and a stats failure should not freeze it.
func TestStatsFailureDoesNotDropBackgroundPollData(t *testing.T) {
	m := GetInitialModel(utils.ComposeSource{})

	containers := []apptypes.DockerContainer{
		{ID: "aaa", Service: "web", State: "running"},
		{ID: "bbb", Service: "db", State: "running"},
	}

	m = updateForTest(t, m, cmds.GetRunningContainersMsg{
		Containers: containers,
		Background: true,
	})

	if !m.waitingForStats {
		t.Fatal("background poll did not arm waitingForStats")
	}

	// docker stats is unavailable, so GetContainerStats hands the poll's
	// containers back unenriched with Err set - the shape asserted by
	// TestGetContainerStatsKeepsContainersOnFailure.
	m = updateForTest(t, m, cmds.GetContainerStatsMsg{
		Containers: containers,
		Err:        errors.New("docker stats failed"),
		Background: true,
	})

	if m.waitingForStats {
		t.Error("waitingForStats stayed armed after a stats failure")
	}

	if m.containers.runningCount != 2 {
		t.Errorf("runningCount = %d, want 2: the poll's containers were dropped when stats failed",
			m.containers.runningCount)
	}
}
