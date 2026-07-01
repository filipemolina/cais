package apptypes

import (
	"stack-stitcher/src/appstyles"

	"charm.land/lipgloss/v2"
)

type ContainerListItem DockerContainer

func (i ContainerListItem) Title() string       { return i.Names }
func (i ContainerListItem) FilterValue() string { return i.Names }
func (i ContainerListItem) Description(isSelected bool) string {
	wrapperStyle := lipgloss.NewStyle()

	if isSelected {
		wrapperStyle = wrapperStyle.Background(appstyles.PanelBackgroundColor)
	}

	boldStyle := wrapperStyle.
		Foreground(appstyles.PrimaryFontColor).
		Bold(true)

	normalStyle := wrapperStyle.Foreground(appstyles.SecondaryFontColor)
	description := boldStyle.Render("Status: ") + normalStyle.Render(i.Status)

	return description
}
