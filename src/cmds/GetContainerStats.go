package cmds

import (
	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

// GetContainerStatsMsg carries containers enriched with runtime stats from
// `docker stats`. Containers is always the full set the poll found, enriched
// when the stats call succeeded and plain when it did not: a background poll
// withholds its own GetRunningContainersMsg from the panels and lets this
// message deliver the data, so dropping it here would freeze every panel
// until the next foreground refresh. Err reports the stats failure for
// callers that care; it never means "no containers".
type GetContainerStatsMsg struct {
	Containers []apptypes.DockerContainer
	Err        error
	Background bool
}

// GetContainerStats fetches docker stats and merges them into the given
// containers slice. Each container is matched by ID.
func GetContainerStats(containers []apptypes.DockerContainer, background bool) tea.Cmd {
	return func() tea.Msg {
		stats, err := utils.DockerStats()
		if err != nil {
			// Stats are an enrichment, not the payload. Hand the containers
			// back unenriched so the panels still get this poll's state.
			return GetContainerStatsMsg{Containers: containers, Err: err, Background: background}
		}

		// Build a lookup map by container ID. Both `docker compose ps` and
		// `docker stats` report the short (12-char) form, so the IDs compare
		// directly.
		statsMap := make(map[string]utils.DockerStatsContainer)
		for _, s := range stats {
			statsMap[s.ID] = s
		}

		enriched := make([]apptypes.DockerContainer, len(containers))
		for i, c := range containers {
			enriched[i] = c
			if stat, ok := statsMap[c.ID]; ok {
				enriched[i].MemPerc = stat.MemPerc
				enriched[i].MemUsage = stat.MemUsage
				enriched[i].NetIO = stat.NetIO
				enriched[i].BlockIO = stat.BlockIO
				enriched[i].CPUPerc = stat.CPUPerc
				enriched[i].PIDs = stat.PIDs
			}
		}

		return GetContainerStatsMsg{Containers: enriched, Background: background}
	}
}
