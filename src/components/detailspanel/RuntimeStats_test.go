package detailspanel

import (
	"strings"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/stack-stitcher/src/apptypes"
)

// panelWithContainer builds a details panel showing one service backed by the
// given container, ready for renderRuntimeStats.
func panelWithContainer(container apptypes.DockerContainer) Model {
	service := types.ServiceConfig{Name: container.Service}

	return Model{
		service:    &service,
		containers: []apptypes.DockerContainer{container},
	}
}

func TestRenderRuntimeStatsSkipsWhenNothingToShow(t *testing.T) {
	cases := []struct {
		name      string
		container apptypes.DockerContainer
	}{
		{
			name:      "no stats and no uptime",
			container: apptypes.DockerContainer{Service: "web", State: "running"},
		},
		{
			name: "container is not running",
			container: apptypes.DockerContainer{
				Service: "web", State: "exited", CPUPerc: "0.00%", MemUsage: "21MiB / 31GiB",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := panelWithContainer(tc.container).renderRuntimeStats(60); got != "" {
				t.Errorf("renderRuntimeStats = %q, want empty", got)
			}
		})
	}
}

// The old guard tested four of the six fields it renders, so a container
// reporting only PIDs - or only the uptime that comes from `docker compose
// ps` rather than `docker stats` - rendered nothing at all.
func TestRenderRuntimeStatsRendersPartialData(t *testing.T) {
	cases := []struct {
		name      string
		container apptypes.DockerContainer
		wantRow   string
	}{
		{
			name: "PIDs only",
			container: apptypes.DockerContainer{
				Service: "web", State: "running", PIDs: "19",
			},
			wantRow: "PIDs",
		},
		{
			name: "uptime only, as when docker stats failed",
			container: apptypes.DockerContainer{
				Service: "web", State: "running", RunningFor: "2 hours ago",
			},
			wantRow: "Uptime",
		},
		{
			name: "the ordinary full set",
			container: apptypes.DockerContainer{
				Service: "web", State: "running",
				MemUsage: "21.71MiB / 31.02GiB", MemPerc: "0.07%",
				CPUPerc: "0.00%", NetIO: "3.22MB / 4.7kB",
				BlockIO: "70.3MB / 43.7MB", PIDs: "19", RunningFor: "2 hours ago",
			},
			wantRow: "Memory",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := panelWithContainer(tc.container).renderRuntimeStats(60)

			if got == "" {
				t.Fatalf("renderRuntimeStats returned nothing for %s", tc.name)
			}
			if !strings.Contains(got, tc.wantRow) {
				t.Errorf("renderRuntimeStats = %q, want a %s row", got, tc.wantRow)
			}
		})
	}
}
