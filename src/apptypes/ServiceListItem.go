package apptypes

import (
	"image/color"
	"strings"

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
	// MemUsage is the real-time memory usage from docker stats, e.g.
	// "21.71MiB / 31.02GiB". Empty when stats are unavailable.
	MemUsage string
	// MemPerc is the memory usage percentage from docker stats, e.g. "0.07%"
	MemPerc string
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

	memLabel := boldStyle.Render("Mem: ")
	var memValue string
	if s.MemUsage != "" {
		memValue = normalStyle.Render(formatMemUsage(s.MemUsage, s.MemPerc))
	} else {
		memValue = normalStyle.Render("—")
	}

	return memLabel + memValue
}

// formatMemUsage formats memory usage as "Usage (Percent)", e.g.,
// "21.71MiB / 31.02GiB" + "0.07%" -> "21.71MiB (0.07%)".
// If percent is empty, returns just the usage part (before "/").
func formatMemUsage(memUsage, memPerc string) string {
	usage := memUsage
	if idx := strings.Index(memUsage, "/"); idx != -1 {
		usage = strings.TrimSpace(memUsage[:idx])
	}
	if memPerc != "" {
		return usage + " (" + memPerc + ")"
	}
	return usage
}
