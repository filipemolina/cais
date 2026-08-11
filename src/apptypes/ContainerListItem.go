package apptypes

import (
	"github.com/filipemolina/cais/src/appstyles"

	"charm.land/lipgloss/v2"
)

type ContainerListItem DockerContainer

func (i ContainerListItem) Title() string       { return i.Names }
func (i ContainerListItem) FilterValue() string { return i.Names }
func (i ContainerListItem) Description(isSelected bool) string {
	wrapperStyle := lipgloss.NewStyle()

	if isSelected {
		wrapperStyle = wrapperStyle.Background(appstyles.Active.ModalBg)
	}

	boldStyle := wrapperStyle.
		Foreground(appstyles.Active.TextPrimary).
		Bold(true)

	normalStyle := wrapperStyle.Foreground(appstyles.Active.TextMuted)
	description := boldStyle.Render("Status: ") + normalStyle.Render(i.Status)

	return description
}
