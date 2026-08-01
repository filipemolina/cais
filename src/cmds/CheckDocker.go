package cmds

import (
	"github.com/filipemolina/stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

// DockerStatusMsg carries the result of a docker preflight probe: the
// startup check, or a re-probe triggered by a docker call's own error - see
// D4 in docs/plans/docker-preflight.md.
type DockerStatusMsg struct{ Status utils.DockerStatus }

// CheckDocker runs the preflight off the update loop. It is under 100ms
// (see docs/plans/docker-preflight.md's Timings), which is why it is cheap
// enough to run both at startup and on every docker error.
func CheckDocker() tea.Cmd {
	return func() tea.Msg {
		return DockerStatusMsg{Status: utils.DockerPreflight()}
	}
}
