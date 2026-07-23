package components

import (
	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/constants"

	"charm.land/lipgloss/v2"
)

// renderActionButtons renders the shared Start/Stop/Restart/Pull/Remove row
// used by both DetailsPanel and GroupDetailsPanel.
func renderActionButtons(width int) string {
	startButton := Button("Start", "s").View().Content
	stopButton := Button("Stop", "t").View().Content
	restartButton := Button("Restart", "r").View().Content
	pullButton := Button("Pull", "p").View().Content
	removeButton := Button("Remove", "x").View().Content

	return lipgloss.NewStyle().
		Width(width - 5).
		AlignHorizontal(lipgloss.Right).
		Render(lipgloss.JoinHorizontal(lipgloss.Left, startButton, stopButton, restartButton, pullButton, removeButton))
}

// renderPanelFrame renders the border/title chrome shared by DetailsPanel
// and GroupDetailsPanel. An empty body renders the idle logo/slogan screen
// instead (nothing selected yet).
func renderPanelFrame(title string, isFocused bool, width int, height int, body string, buttons string) string {
	style := wrapperStyle.
		Width(width).
		Height(height - 1)

	if isFocused {
		style = style.
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(appstyles.PrimaryColor).
			Padding(0, 1)
	}

	titleRendered := appstyles.NormalTitle.Render(title)

	if body == "" {
		screen := lipgloss.JoinVertical(lipgloss.Left, constants.LOGO, "", "", constants.SLOGAN)

		return style.
			Align(lipgloss.Center).
			AlignVertical(lipgloss.Center).
			Render(lipgloss.JoinVertical(lipgloss.Left, titleRendered, screen))
	}

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, titleRendered, buttons, body))
}
