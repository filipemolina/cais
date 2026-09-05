package chrome

import (
	"fmt"
	"image/color"

	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"

	"github.com/filipemolina/cais/src/appstyles"
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

// RenderPendingAction renders a spinner with the action description, centered
// on the panel's footer row, while a docker action is in progress.
//
// Both details panels drew this identically. It lives here so the two footers
// cannot drift apart the way the "is this service running" answers once did.
func RenderPendingAction(s spinner.Model, action *PendingAction, width int, bg color.Color) string {
	desc := ActionDescription(action.Action, action.Target, action.IsGroup)

	style := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextPrimary).
		Background(bg).
		Width(width).
		AlignHorizontal(lipgloss.Center)

	return style.Render(s.View() + " " + desc)
}
