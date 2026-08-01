package cmds

import (
	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

// DockerUsageMsg carries the disk usage and memory total from `docker system df`
// and `docker info`. If Err is set, the fetch failed and the overlay should
// show the error (or defer to the preflight diagnosis).
type DockerUsageMsg struct {
	Disk       []utils.DiskUsage
	MemTotal   int64
	Containers []apptypes.DockerContainer // passed in from AppModel to sum memory
	Err        error
}

// GetDockerUsage runs `docker system df --format json` and `docker info --format '{{.MemTotal}}'`
// and returns a DockerUsageMsg. The containers slice is passed in so the command can
// sum the used memory (from MemUsage) without the caller having to do it.
func GetDockerUsage(containers []apptypes.DockerContainer) tea.Cmd {
	return func() tea.Msg {
		dfOutput, err := utils.DockerSystemDf()
		if err != nil {
			return DockerUsageMsg{Err: err}
		}

		disk, err := utils.ParseSystemDf(dfOutput)
		if err != nil {
			return DockerUsageMsg{Err: err}
		}

		memTotal, err := utils.DockerMemTotal()
		if err != nil {
			// Not fatal: we can fall back to the limit side of MemUsage strings.
			memTotal = 0
		}

		return DockerUsageMsg{
			Disk:       disk,
			MemTotal:   memTotal,
			Containers: containers,
		}
	}
}