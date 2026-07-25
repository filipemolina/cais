package components

import (
	"stack-stitcher/src/appstyles"

	"charm.land/lipgloss/v2"
)

// renderActionButtons renders the shared Start/Stop/Restart/Pull/Remove row
// used by both DetailsPanel and GroupDetailsPanel, right-aligned within the
// panel body width it is given.
func renderActionButtons(width int) string {
	startButton := Button("Start", "s").View().Content
	stopButton := Button("Stop", "t").View().Content
	restartButton := Button("Restart", "r").View().Content
	pullButton := Button("Pull", "p").View().Content
	removeButton := Button("Remove", "x").View().Content

	return lipgloss.NewStyle().
		Width(max(0, width)).
		AlignHorizontal(lipgloss.Right).
		Render(lipgloss.JoinHorizontal(lipgloss.Left, startButton, stopButton, restartButton, pullButton, removeButton))
}

// renderEmptyCard renders a dim, centered, rounded-border card used for the
// empty / onboarding states. `key` is shown in the accent color inside
// brackets, `hint` is the trailing description in a dim color. `availHeight`
// is the vertical space in which the card should be centered.
func renderEmptyCard(width, availHeight int, title, body, key, hint string) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(appstyles.TextMuted)
	bodyStyle := lipgloss.NewStyle().Foreground(appstyles.TextDim)
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(appstyles.Accent)
	hintStyle := lipgloss.NewStyle().Foreground(appstyles.TextDim)

	content := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(title),
		"",
		bodyStyle.Render(body),
	)

	if hint != "" {
		hintLine := keyStyle.Render("["+key+"]") + hintStyle.Render(" "+hint)
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", hintLine)
	}

	cardWidth := width - 8
	if cardWidth > 54 {
		cardWidth = 54
	}
	if cardWidth < 20 {
		cardWidth = 20
	}

	card := lipgloss.NewStyle().
		Width(cardWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(appstyles.BorderDefault).
		AlignHorizontal(lipgloss.Center).
		Render(content)

	if availHeight < 1 {
		availHeight = 1
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(availHeight).
		MaxHeight(availHeight).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Render(card)
}

// renderPanelFrame renders the title chrome shared by DetailsPanel and
// GroupDetailsPanel, filling exactly the width x height box the panel was
// given. The 3-tier background system handles focus: tier 3 (panel) when
// unfocused, tier 4 (elevated) when focused.
//
// Callers embed their action buttons at the bottom of `body`, which pins the
// action row to the bottom of the panel.
func renderPanelFrame(title string, isFocused bool, width int, height int, body string) string {
	bg := appstyles.BackgroundPanel
	if isFocused {
		bg = appstyles.BackgroundElevated
	}

	style := fitBox(wrapperStyle.Background(bg), width, height)
	titleRendered := appstyles.NormalTitle.Render(title)

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, titleRendered, body))
}

// panelBodyWidth and panelBodyHeight are the space a panel body gets inside a
// frame of the given total size: the frame's own padding taken off, plus the
// title row for the vertical axis. Callers size their content with these
// rather than with hardcoded offsets, so a change to wrapperStyle's padding
// doesn't silently push content out of the panel.
func panelBodyWidth(total int) int {
	frameW, _ := wrapperStyle.GetFrameSize()

	return max(0, total-frameW)
}

func panelBodyHeight(total int) int {
	_, frameH := wrapperStyle.GetFrameSize()

	return max(0, total-frameH-1) // -1 for the title row
}
