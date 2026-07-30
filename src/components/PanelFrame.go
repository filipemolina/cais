package components

import (
	"image/color"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

// dockerActionFor returns the `docker compose` action a keypress asks for.
// Both details panels read it, which is what makes "s starts a group" and
// "s starts a service" the same fact rather than two switch statements that
// happen to agree. Remove is absent on purpose: it is destructive, so it goes
// through a confirmation instead of straight to a command.
func dockerActionFor(msg tea.KeyPressMsg) (string, bool) {
	switch {
	case key.Matches(msg, keys.Details.Start):
		return "start", true
	case key.Matches(msg, keys.Details.Stop):
		return "stop", true
	case key.Matches(msg, keys.Details.Restart):
		return "restart", true
	case key.Matches(msg, keys.Details.Pull):
		return "pull", true
	}

	return "", false
}

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
		BorderForeground(appstyles.Active.Accent).
		BorderBackground(bg).
		Background(bg)

	return appstyles.FillBackground(bg, style.Render(content))
}

// modalTitle renders a modal's heading. Every modal names itself, so a user
// who lands on one mid-flow can tell what it is about to do without having to
// infer it from the fields. Accent + bold is the same treatment the list-based
// modals get from list.Styles.Title, so a titled input modal and a titled
// picker read as the same kind of surface.
//
// The margin is part of the title rather than a blank line each caller
// remembers to add, and it matches the blank row the hint line sits above: the
// heading and the footer are the modal's chrome, and both stand off the body.
func modalTitle(text string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(appstyles.Active.Accent).
		Background(appstyles.Active.ModalBg).
		MarginBottom(1).
		Render(text)
}

// modalHints renders a modal's own help line, in the footer bar's format but
// with the lighter description color the modal surface needs. Every modal
// carries one: the footer bar is hidden behind the modal while it is open, so
// the keys the modal takes over are advertised here or nowhere.
func modalHints(hints ...KeyHint) string {
	return renderKeyHints(hints, appstyles.Active.TextMuted)
}

// renderEmptyCard renders a dim, centered, rounded-border card used for the
// empty / onboarding states. `key` is shown in the accent color inside
// brackets, `hint` is the trailing description in a dim color. `availHeight`
// is the vertical space in which the card should be centered. `bg` is the
// panel's background tier, which the space around the card sits on; the card
// itself is recessed below that tier so it reads as inset into the panel.
func renderEmptyCard(width, availHeight int, bg color.Color, title, body, key, hint string) string {
	cardBg := appstyles.Active.BackgroundRecessed

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(appstyles.Active.TextMuted).Background(cardBg)
	bodyStyle := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).Background(cardBg)
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(appstyles.Active.Accent).Background(cardBg)
	hintStyle := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).Background(cardBg)

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
		BorderForeground(appstyles.Active.BorderCard).
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
	titleRow := appstyles.NormalTitle().Render(title)

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
