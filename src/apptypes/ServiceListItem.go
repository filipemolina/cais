package apptypes

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/cais/src/appstyles"
)

type ServiceListItem struct {
	Service types.ServiceConfig
	// Status reflects the docker container state for this service:
	// "running", "stopped", or "" (unknown / no container yet). Set by
	// ServicesListModel when a GetRunningContainersMsg arrives.
	Status string
	// MemUsage is the real-time memory usage from docker stats, exactly as
	// docker reports it: "21.71MiB / 31.02GiB". Empty when stats are
	// unavailable. Store it raw - FormatMemUsage is applied once, at render
	// time, and formatting it on the way in instead would double-apply the
	// percent suffix.
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
	var bg color.Color

	switch s.Status {
	case "running":
		label, bg = "RUNNING", appstyles.Active.StatusRunning
	default:
		label, bg = "STOPPED", appstyles.Active.StatusStopped
	}
	fg := appstyles.InkOn(bg)

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
		memValue = normalStyle.Render(FormatMemUsage(s.MemUsage, s.MemPerc))
	} else {
		memValue = normalStyle.Render("—")
	}

	return memLabel + memValue
}

// FormatMemUsage formats memory usage as "Usage (Percent)", e.g.,
// "21.71MiB / 31.02GiB" + "0.07%" -> "21.71MiB (0.07%)".
// If percent is empty, returns just the usage part (before "/").
//
// It takes docker's raw strings, and is the one copy: the service list row
// and the details panel's runtime table both render through it, so the two
// never drift apart. Applying it twice is not idempotent - the second pass
// finds no "/" to split on and appends the percent again - so call it at
// render time and keep the raw values in the fields.
func FormatMemUsage(memUsage, memPerc string) string {
	usage := memUsage
	if idx := strings.Index(memUsage, "/"); idx != -1 {
		usage = strings.TrimSpace(memUsage[:idx])
	}
	if memPerc != "" {
		return usage + " (" + memPerc + ")"
	}
	return usage
}
