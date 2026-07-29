package cmds

import (
	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

// GetContainerStatsMsg carries containers enriched with runtime stats from
// `docker stats`. On error, Containers may be nil.
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
			return GetContainerStatsMsg{Err: err, Background: background}
		}

		// Build a lookup map by container ID (first 12 chars match).
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
