package cmds

import (
	"github.com/filipemolina/cais/src/utils"

	tea "charm.land/bubbletea/v2"
)

// CycleRestartPolicyMsg reports the result of advancing a service's
// restart: policy to the next value in the cycle.
type CycleRestartPolicyMsg struct {
	ServiceName string
	Policy      string
	Err         error
}

// CycleRestartPolicyRequestMsg asks AppModel to advance serviceName's
// restart: policy. The details panel emits this instead of the command
// itself: like every other write-path action, it has no business knowing
// which compose file is loaded or what the service's current policy is -
// both live on AppModel's loaded config.
type CycleRestartPolicyRequestMsg struct {
	ServiceName string
}

// RequestCycleRestartPolicy asks AppModel to cycle serviceName's restart
// policy.
func RequestCycleRestartPolicy(serviceName string) tea.Cmd {
	return func() tea.Msg {
		return CycleRestartPolicyRequestMsg{ServiceName: serviceName}
	}
}

// CycleRestartPolicy advances serviceName from current to the next policy
// in utils.NextRestartPolicy's cycle, in the compose file at fileName.
func CycleRestartPolicy(fileName, serviceName, current string) tea.Cmd {
	return func() tea.Msg {
		next := utils.NextRestartPolicy(current)
		return CycleRestartPolicyMsg{
			ServiceName: serviceName,
			Policy:      next,
			Err:         utils.SetRestartPolicy(fileName, serviceName, next),
		}
	}
}
