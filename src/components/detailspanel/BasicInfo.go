package detailspanel

import (
	"fmt"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/components/chrome"

	"charm.land/lipgloss/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

// basicInfo renders the selected service's summary card at exactly `width`
// columns (border included), so the card spans the details panel instead of
// shrinking to the length of its longest line.
func basicInfo(service types.ServiceConfig, width int) string {
	wrapper := chrome.FitBox(lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(appstyles.Active.Accent).
		Padding(1), width, 0)

	nameHeader := lipgloss.NewStyle().Bold(true).Render("Name: ")
	puidHeader := lipgloss.NewStyle().Bold(true).Render("PUID: ")
	pgidHeader := lipgloss.NewStyle().Bold(true).Render("PGID: ")
	imageHeader := lipgloss.NewStyle().Bold(true).Render("Image: ")
	portsHeader := lipgloss.NewStyle().Bold(true).Render("Ports: ")
	groupsHeader := lipgloss.NewStyle().Bold(true).Render("Groups: ")
	// volumesHeader := lipgloss.NewStyle().Bold(true).Render("Volumes: ")

	ports := service.Ports
	var portLines []string

	for _, port := range ports {
		protocol := port.Protocol
		if protocol == "" {
			protocol = "tcp"
		}

		portLine := fmt.Sprintf("%d/%s", port.Target, protocol)
		if port.Published != "" {
			portLine = port.Published + "->" + portLine
		}

		portLines = append(portLines, portLine)
	}

	portContent := lipgloss.JoinVertical(lipgloss.Left, portLines...)
	var puid string
	var pgid string

	if value, ok := service.Environment["PUID"]; ok {
		puid = *value
	}

	if value, ok := service.Environment["PGID"]; ok {
		pgid = *value
	}

	nameLine := lipgloss.JoinHorizontal(lipgloss.Top, nameHeader, service.Name)
	imageLine := lipgloss.JoinHorizontal(lipgloss.Top, imageHeader, service.Image)
	groupsLine := lipgloss.JoinHorizontal(lipgloss.Top, groupsHeader, fmt.Sprintf("%+v", service.Profiles))

	// PUID/PGID are optional, env-var-derived fields (common in the *arr
	// stack, absent for most services). Show the row only when at least one
	// is set, so the card never carries empty "PUID:  PGID: " labels.
	rows := []string{nameLine}
	if puid != "" || pgid != "" {
		var idParts []string
		if puid != "" {
			idParts = append(idParts, puidHeader, puid)
		}
		if pgid != "" {
			if puid != "" {
				idParts = append(idParts, "  ", pgidHeader, pgid)
			} else {
				idParts = append(idParts, pgidHeader, pgid)
			}
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, idParts...))
	}
	rows = append(rows, imageLine, groupsLine, portsHeader, portContent)

	info := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return wrapper.Render(info)
}
