package chrome

import (
	"fmt"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/filipemolina/stack-stitcher/src/appstyles"
)

// PendingAction tracks a docker action that is currently running.
type PendingAction struct {
	Action  string
	Target  string
	IsGroup bool
}

// NewSpinner creates a spinner styled with the active theme's accent color.
func NewSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(appstyles.Active.Accent)
	return s
}

// actionLabel returns a human-readable label for the action.
func actionLabel(action string) string {
	switch action {
	case "start":
		return "Starting"
	case "stop":
		return "Stopping"
	case "restart":
		return "Restarting"
	case "pull":
		return "Pulling"
	case "remove":
		return "Removing"
	default:
		return "Running"
	}
}

// kindLabel returns "group" or "service" for display.
func kindLabel(isGroup bool) string {
	if isGroup {
		return "group"
	}
	return "service"
}

// ActionDescription returns a full description of the pending action.
func ActionDescription(action, target string, isGroup bool) string {
	return fmt.Sprintf("%s %s %q...", actionLabel(action), kindLabel(isGroup), target)
}

// HandleSpinnerTick updates the spinner and returns the next tick command.
// Returns nil if no spinner is active.
func HandleSpinnerTick(spinnerModel spinner.Model, pendingAction *PendingAction, msg tea.Msg) (spinner.Model, tea.Cmd) {
	if pendingAction == nil {
		return spinnerModel, nil
	}

	var cmd tea.Cmd
	spinnerModel, cmd = spinnerModel.Update(msg)
	return spinnerModel, cmd
}
