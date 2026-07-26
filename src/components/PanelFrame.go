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

// modalSurface wraps a modal's content in the shared modal chrome: an accent
// rounded border, padding, and a background sealed against `bg` so the modal
// reads as one opaque surface over the page it is composited onto. Modals in
// particular cannot afford an unpainted cell - the page shows through it.
//
// BorderBackground is set explicitly because lipgloss leaves border cells on
// the default background otherwise, which outlines the modal in the terminal's
// color.
func modalSurface(bg color.Color, content string) string {
	style := lipgloss.NewStyle().
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(appstyles.PrimaryColor).
		BorderBackground(bg).
		Background(bg)

	return appstyles.FillBackground(bg, style.Render(content))
}

// renderEmptyCard renders a dim, centered, rounded-border card used for the
// empty / onboarding states. `key` is shown in the accent color inside
// brackets, `hint` is the trailing description in a dim color. `availHeight`
// is the vertical space in which the card should be centered. `bg` is the
// panel's background tier, which the space around the card sits on; the card
// itself is recessed below that tier so it reads as inset into the panel.
func renderEmptyCard(width, availHeight int, bg color.Color, title, body, key, hint string) string {
	cardBg := appstyles.BackgroundRecessed

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(appstyles.TextMuted).Background(cardBg)
	bodyStyle := lipgloss.NewStyle().Foreground(appstyles.TextDim).Background(cardBg)
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(appstyles.Accent).Background(cardBg)
	hintStyle := lipgloss.NewStyle().Foreground(appstyles.TextDim).Background(cardBg)

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

	// The border ring carries the card's own background, so the recessed
	// surface reads as one solid block rather than a dark fill outlined in the
	// panel color.
	card := lipgloss.NewStyle().
		Width(cardWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(appstyles.BorderCard).
		BorderBackground(cardBg).
		Background(cardBg).
		AlignHorizontal(lipgloss.Center).
		Render(content)

	if availHeight < 1 {
		availHeight = 1
	}

	// The card's inner joins leave unstyled padding on the short lines (the
	// blank spacer rows, and the hint line built by concatenating two styled
	// runs). Seal against the card's surface, not the panel's - this runs
	// before the panel seals the space around it.
	card = appstyles.FillBackground(cardBg, card)

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
// `titleRight` is an optional accessory pinned to the right end of the title
// row - the group status pill uses it. A blank row separates the title row
// from the body, so the panel's own label doesn't read as part of whatever
// the body starts with; panelBodyHeight accounts for both rows.
//
// Callers embed their action buttons at the bottom of `body`, which pins the
// action row to the bottom of the panel.
func renderPanelFrame(title string, titleRight string, isFocused bool, width int, height int, body string) string {
	bg := panelBg(isFocused)

	style := fitBox(wrapperStyle.Background(bg), width, height)
	titleRow := appstyles.NormalTitle.Render(title)

	if titleRight != "" {
		gap := max(0, panelBodyWidth(width)-lipgloss.Width(titleRow)-lipgloss.Width(titleRight))

		titleRow = lipgloss.JoinHorizontal(lipgloss.Top,
			titleRow,
			lipgloss.NewStyle().Background(bg).Width(gap).Render(""),
			titleRight,
		)
	}

	// The panel is where tier 3/4 is established, so it is where the tier's
	// background has to be sealed in. JoinVertical pads the short title row out
	// to the body width with unstyled spaces, and the body itself arrives with
	// whatever gaps its own joins left; FillBackground closes both. Repainting
	// before fitBox's Width() padding is applied is fine - that padding is
	// styled by lipgloss already.
	content := appstyles.FillBackground(bg, lipgloss.JoinVertical(lipgloss.Left, titleRow, "", body))

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

	return max(0, total-frameH-2) // -2 for the title row and the blank row under it
}
