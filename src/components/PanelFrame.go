package components

import (
	"image/color"
	"stack-stitcher/src/appstyles"

	"charm.land/lipgloss/v2"
)

// renderActionButtons renders the shared Start/Stop/Restart/Pull/Remove row
// used by both DetailsPanel and GroupDetailsPanel, right-aligned within the
// panel body width it is given. `bg` is the panel's background tier, which the
// buttons sit flush on.
func renderActionButtons(width int, bg color.Color) string {
	startButton := Button("Start", "s", bg).View().Content
	stopButton := Button("Stop", "t", bg).View().Content
	restartButton := Button("Restart", "r", bg).View().Content
	pullButton := Button("Pull", "p", bg).View().Content
	removeButton := Button("Remove", "x", bg).View().Content

	row := lipgloss.JoinHorizontal(lipgloss.Left, startButton, stopButton, restartButton, pullButton, removeButton)

	// JoinHorizontal pads each button up to the tallest one with unstyled
	// spaces, which is the dark band that used to sit behind this row.
	row = appstyles.FillBackground(bg, row)

	return lipgloss.NewStyle().
		Width(max(0, width)).
		AlignHorizontal(lipgloss.Right).
		Background(bg).
		Render(row)
}

// renderEmptyCard renders a dim, centered, rounded-border card used for the
// empty / onboarding states. `key` is shown in the accent color inside
// brackets, `hint` is the trailing description in a dim color. `availHeight`
// is the vertical space in which the card should be centered. `bg` is the
// panel's background tier, which both the card and the space around it sit on.
func renderEmptyCard(width, availHeight int, bg color.Color, title, body, key, hint string) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(appstyles.TextMuted).Background(bg)
	bodyStyle := lipgloss.NewStyle().Foreground(appstyles.TextDim).Background(bg)
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(appstyles.Accent).Background(bg)
	hintStyle := lipgloss.NewStyle().Foreground(appstyles.TextDim).Background(bg)

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
		BorderBackground(bg).
		Background(bg).
		AlignHorizontal(lipgloss.Center).
		Render(content)

	if availHeight < 1 {
		availHeight = 1
	}

	// The card's inner joins leave unstyled padding on the short lines (the
	// blank spacer rows, and the hint line built by concatenating two styled
	// runs), which is the dark block that used to sit beside the card.
	card = appstyles.FillBackground(bg, card)

	return lipgloss.NewStyle().
		Width(width).
		Height(availHeight).
		MaxHeight(availHeight).
		Background(bg).
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
	bg := panelBg(isFocused)

	style := fitBox(wrapperStyle.Background(bg), width, height)
	titleRendered := appstyles.NormalTitle.Render(title)

	// The panel is where tier 3/4 is established, so it is where the tier's
	// background has to be sealed in. JoinVertical pads the short title row out
	// to the body width with unstyled spaces, and the body itself arrives with
	// whatever gaps its own joins left; FillBackground closes both. Repainting
	// before fitBox's Width() padding is applied is fine - that padding is
	// styled by lipgloss already.
	content := appstyles.FillBackground(bg, lipgloss.JoinVertical(lipgloss.Left, titleRendered, body))

	return style.Render(content)
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
