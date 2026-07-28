package cmds

import tea "charm.land/bubbletea/v2"

// SetPendingActionMsg signals that a docker action is in progress.
// The panels show a spinner and disable action keys until it completes.
type SetPendingActionMsg struct {
	Action  string
	Target  string
	IsGroup bool
}

// SetPendingAction returns a command that announces a docker action is starting.
func SetPendingAction(action string, target string, isGroup bool) tea.Cmd {
	return func() tea.Msg {
		return SetPendingActionMsg{Action: action, Target: target, IsGroup: isGroup}
	}
}

// ClearPendingActionMsg signals that the pending docker action has completed.
type ClearPendingActionMsg struct{}

// ClearPendingAction returns a command that announces the pending action is done.
func ClearPendingAction() tea.Cmd {
	return func() tea.Msg { return ClearPendingActionMsg{} }
}
