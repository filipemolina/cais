package components

import (
	"fmt"
	"stack-stitcher/src/appstyles"

	"charm.land/lipgloss/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

func BasicInfo(service types.ServiceConfig, width int) string {
	wrapper := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(appstyles.PrimaryColor).
		Padding(1)

	nameHeader := lipgloss.NewStyle().Bold(true).Render("Name: ")
	puidHeader := lipgloss.NewStyle().Bold(true).Render("PUID: ")
	pgidHeader := lipgloss.NewStyle().Bold(true).Render(" PGID: ")
	imageHeader := lipgloss.NewStyle().Bold(true).Render("Image: ")
	portsHeader := lipgloss.NewStyle().Bold(true).Render("Ports: ")
	profilesHeader := lipgloss.NewStyle().Bold(true).Render("Profiles: ")
	// volumesHeader := lipgloss.NewStyle().Bold(true).Render("Volumes: ")

	ports := service.Ports
	var portLines []string

	for _, port := range ports {
		// portLines = append(portLines, port.Published+"\\"+strconv.FormatUint(uint64(port.Target), 100))
		portLines = append(portLines, fmt.Sprintf("%+v", port))
	}

	portContent := lipgloss.JoinVertical(lipgloss.Left, portLines...)
	var puid string
	var pgid string

	if value, ok := service.Environment["PUID"]; ok {
		puid = *value
	}

	if value, ok := service.Environment["PUID"]; ok {
		pgid = *value
	}

	nameLine := lipgloss.JoinHorizontal(lipgloss.Top, nameHeader, service.ContainerName)
	idLine := lipgloss.JoinHorizontal(lipgloss.Top, puidHeader, puid, pgidHeader, pgid)
	imageLine := lipgloss.JoinHorizontal(lipgloss.Top, imageHeader, service.Image)
	profilesLine := lipgloss.JoinHorizontal(lipgloss.Top, profilesHeader, fmt.Sprintf("%+v", service.Profiles))

	info := lipgloss.JoinVertical(lipgloss.Left, nameLine, idLine, imageLine, portsHeader, profilesLine, portContent)

	return wrapper.Render(info)
}
