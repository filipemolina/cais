package cmds

import (
	"github.com/filipemolina/stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

type DockerActionMsg struct {
	Action  string
	Target  string
	IsGroup bool
	Err     error
}

// RunDockerActionMsg asks AppModel to run a docker action. Panels emit this
// rather than the command itself, for the same reason they emit OpenEditorMsg:
// the action has to run against the compose file the app resolved, and that is
// AppModel's state, not a panel's.
type RunDockerActionMsg struct {
	Action  string
	Target  string
	IsGroup bool
}

// RequestDockerAction asks AppModel to run action against target. Returning a
// command (rather than the message) keeps it usable as a ConfirmModal's
// on-confirm command.
func RequestDockerAction(action string, target string, isGroup bool) tea.Cmd {
	return func() tea.Msg {
		return RunDockerActionMsg{Action: action, Target: target, IsGroup: isGroup}
	}
}

// RunDockerAction runs a docker compose action (start, stop, restart, pull,
// remove) against a single service or every service in a group, off the
// main Update loop. composeFile scopes it to the file the app has loaded.
func RunDockerAction(action string, target string, isGroup bool, composeFile string) tea.Cmd {
	return func() tea.Msg {
		err := utils.RunDockerCompose(action, target, isGroup, composeFile)

		return DockerActionMsg{
			Action:  action,
			Target:  target,
			IsGroup: isGroup,
			Err:     err,
		}
	}
}
