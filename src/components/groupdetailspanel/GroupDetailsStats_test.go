package groupdetailspanel

import (
	"testing"

	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/cmds"
)

// On Home, a background poll's GetRunningContainersMsg is withheld from the
// panels while stats are in flight; the enriched GetContainerStatsMsg is what
// arrives instead. New renders per-service running state, so it
// has to accept that message too or its rows go stale between foreground
// refreshes.
func TestGroupDetailsPanelAcceptsContainerStats(t *testing.T) {
	panel := New()

	updated, _ := panel.Update(cmds.GetContainerStatsMsg{
		Containers: []apptypes.DockerContainer{
			{ID: "aaa", Service: "web", State: "running"},
		},
		Background: true,
	})

	model, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", updated)
	}

	if len(model.containers) != 1 {
		t.Fatalf("containers = %d, want 1: GetContainerStatsMsg was ignored", len(model.containers))
	}

	if !model.isServiceRunning("web") {
		t.Error("web should read as running after a stats message")
	}
}
