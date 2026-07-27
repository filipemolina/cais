package apptypes

import (
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"strconv"

	"charm.land/lipgloss/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

type ServiceListItem struct {
	Service types.ServiceConfig
}

func (s ServiceListItem) Title() string       { return s.Service.Name }
func (s ServiceListItem) FilterValue() string { return s.Service.Name }
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
