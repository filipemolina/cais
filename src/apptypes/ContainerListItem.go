package apptypes

import (
	"stack-stitcher/src/appstyles"

	"charm.land/lipgloss/v2"
)

type ContainerListItem DockerContainer

func (i ContainerListItem) Title() string       { return i.Names }
func (i ContainerListItem) FilterValue() string { return i.Names }
func (i ContainerListItem) Description() string {
	boldStyle := lipgloss.NewStyle().
		Foreground(appstyles.PrimaryFontColor).
		Bold(true)

	imageHeader := boldStyle.Render(" Image: ")
	statusHeader := boldStyle.Render("Status: ")

	description := statusHeader + i.Status + imageHeader + i.Image
	return description
}
