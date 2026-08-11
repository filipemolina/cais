package cmds

import (
	"testing"

	"github.com/filipemolina/cais/src/apptypes"
)

// A background poll withholds its own GetRunningContainersMsg from the panels
// and lets GetContainerStats deliver the containers instead. So a stats
// failure must still carry them: returning nil here freezes every panel until
// the next foreground refresh.
func TestGetContainerStatsKeepsContainersOnFailure(t *testing.T) {
	// An empty PATH makes the `docker` lookup fail, which is the failure the
	// command has to survive.
	t.Setenv("PATH", "")

	containers := []apptypes.DockerContainer{
		{ID: "aaa", Service: "web", State: "running"},
		{ID: "bbb", Service: "db", State: "exited"},
	}

	msg, ok := GetContainerStats(containers, true)().(GetContainerStatsMsg)
	if !ok {
		t.Fatal("GetContainerStats did not return a GetContainerStatsMsg")
	}

	if msg.Err == nil {
		t.Fatal("expected a stats error with docker off the PATH")
	}

	if len(msg.Containers) != len(containers) {
		t.Fatalf("Containers = %d, want %d: the poll's containers were dropped",
			len(msg.Containers), len(containers))
	}

	if msg.Containers[0].Service != "web" || msg.Containers[1].State != "exited" {
		t.Error("containers came back altered by a failed stats call")
	}

	if !msg.Background {
		t.Error("the background flag was lost on the failure path")
	}
}
