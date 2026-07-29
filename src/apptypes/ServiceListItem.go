package apptypes

import (
	"image/color"
	"strconv"

	"charm.land/lipgloss/v2"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
)

type ServiceListItem struct {
	Service types.ServiceConfig
	// Status reflects the docker container state for this service:
	// "running", "stopped", or "" (unknown / no container yet). Set by
	// ServicesListModel when a GetRunningContainersMsg arrives.
	Status string
}

func (s ServiceListItem) Title() string       { return s.Service.Name }
func (s ServiceListItem) FilterValue() string { return s.Service.Name }

// StatusPill returns a styled pill showing the service's current status.
// It uses the same visual language as the group's statusPill in
// GroupDetailsPanel, but for a single service.
func (s ServiceListItem) StatusPill() string {
	var label string
	var bg, fg color.Color

	switch s.Status {
	case "running":
		label, bg, fg = "RUNNING", appstyles.Active.StatusRunning, appstyles.Active.InkOnLight
	default:
		label, bg, fg = "STOPPED", appstyles.Active.StatusStopped, appstyles.Active.InkOnDark
	}

	return lipgloss.NewStyle().
		Background(bg).
		Foreground(fg).
		Bold(true).
		Padding(0, 1).
		Render(label)
}

func (s ServiceListItem) Description(isActive bool) string {
	wrapperStyle := lipgloss.NewStyle()

	if isActive {
		wrapperStyle = wrapperStyle.Background(appstyles.Active.ModalBg)
	}

	boldStyle := wrapperStyle.
		Foreground(appstyles.Active.TextPrimary).
		Bold(true)

	normalStyle := wrapperStyle.Foreground(appstyles.Active.TextMuted)

	cpuUsage := strconv.FormatFloat(float64(s.Service.CPUPercent), 'f', 1, 32)

	description := boldStyle.Render("CPU: ") +
		normalStyle.Render(cpuUsage)

	return description
}
