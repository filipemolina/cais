package components

import (
	"fmt"
	"github.com/filipemolina/stack-stitcher/src/appstyles"

	"charm.land/lipgloss/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

// BasicInfo renders the selected service's summary card at exactly `width`
// columns (border included), so the card spans the details panel instead of
// shrinking to the length of its longest line.
func BasicInfo(service types.ServiceConfig, width int) string {
	wrapper := fitBox(lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(appstyles.Active.Accent).
		Padding(1), width, 0)

	nameHeader := lipgloss.NewStyle().Bold(true).Render("Name: ")
	puidHeader := lipgloss.NewStyle().Bold(true).Render("PUID: ")
	pgidHeader := lipgloss.NewStyle().Bold(true).Render(" PGID: ")
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

	nameLine := lipgloss.JoinHorizontal(lipgloss.Top, nameHeader, service.ContainerName)
	idLine := lipgloss.JoinHorizontal(lipgloss.Top, puidHeader, puid, pgidHeader, pgid)
	imageLine := lipgloss.JoinHorizontal(lipgloss.Top, imageHeader, service.Image)
	groupsLine := lipgloss.JoinHorizontal(lipgloss.Top, groupsHeader, fmt.Sprintf("%+v", service.Profiles))

	info := lipgloss.JoinVertical(lipgloss.Left, nameLine, idLine, imageLine, groupsLine, portsHeader, portContent)

	return wrapper.Render(info)
}
