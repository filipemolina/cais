package cmds

import (
	"stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

type DockerActionMsg struct {
	Action    string
	Target    string
	IsProfile bool
	Err       error
}

// RunDockerAction runs a docker compose action (start, stop, restart, pull,
// remove) against a single service or every service in a profile, off the
// main Update loop.
func RunDockerAction(action string, target string, isProfile bool) tea.Cmd {
	return func() tea.Msg {
		err := utils.RunDockerCompose(action, target, isProfile)

		return DockerActionMsg{
			Action:    action,
			Target:    target,
			IsProfile: isProfile,
			Err:       err,
		}
	}
}
