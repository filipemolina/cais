package chrome

import (
	"image/color"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/keys"
)

// DockerActionFor returns the `docker compose` action a keypress asks for.
// Both details panels read it, which is what makes "s starts a group" and
// "s starts a service" the same fact rather than two switch statements that
// happen to agree. Remove is absent on purpose: it is destructive, so it goes
// through a confirmation instead of straight to a command.
func DockerActionFor(msg tea.KeyPressMsg) (string, bool) {
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

// PanelRule is the thin horizontal line both details panels separate their
// sections with. One helper rather than five copies of the same three-line
// style, so the panels cannot drift to different rules.
func PanelRule(width int) string {
	return lipgloss.NewStyle().
		Foreground(appstyles.Active.BorderDefault).
		Width(width).
		Render(strings.Repeat("─", max(width, 0)))
}

// PanelBodyWithFooter lays out a details panel's body: `content` at the top,
// `footer` on the body's last rows, and blank rows between them. Both details
// panels build their body through here, which is what makes the pending-action
// spinner land on the same line of both rather than wherever each panel's
// content happens to end - see "The panel footer" in docs/DESIGN.md.
//
// The content is clipped before the footer is attached, so a panel whose
// content outgrows its box loses its last rows rather than its footer. Doing
// it the other way round - joining first and clipping the result - takes the
// bottom off, which is exactly the row that has to survive.
//
// An empty footer costs no rows. lipgloss.Height("") is 1, so passing "" would
// otherwise reserve a blank line at the foot of every panel that has nothing
// to pin there - which is most frames, now that the panels only use the footer
// for the spinner and the service panel's transient confirmations.
func PanelBodyWithFooter(width, avail int, bg color.Color, content, footer string) string {
	footerHeight := 0
	if footer != "" {
		footerHeight = lipgloss.Height(footer)
	}

	contentAvail := max(0, avail-footerHeight)

	parts := make([]string, 0, 3)
	if contentAvail > 0 {
		content = lipgloss.NewStyle().MaxHeight(contentAvail).Render(content)
		parts = append(parts, content)

		if gap := contentAvail - lipgloss.Height(content); gap > 0 {
			parts = append(parts, lipgloss.NewStyle().
				Background(bg).
				Width(max(0, width)).
				Height(gap).
				Render(""))
		}
	}
	if footer != "" {
		parts = append(parts, footer)
	}

	body := lipgloss.JoinVertical(lipgloss.Left, parts...)

	// Safety cap: a miscount must never grow the panel past its body region.
	if avail > 0 {
		body = lipgloss.NewStyle().MaxHeight(avail).Render(body)
	}

	return body
}

// ModalSurface wraps a modal's content in the shared modal chrome: an accent
// rounded border, padding, and a background sealed against `bg` so the modal
// reads as one opaque surface over the page it is composited onto. Modals in
// particular cannot afford an unpainted cell - the page shows through it.
//
// BorderBackground is set explicitly because lipgloss leaves border cells on
// the default background otherwise, which outlines the modal in the terminal's
// color.
func ModalSurface(bg color.Color, content string) string {
	style := lipgloss.NewStyle().
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(appstyles.Active.Accent).
		BorderBackground(bg).
		Background(bg)

	return appstyles.FillBackground(bg, style.Render(content))
}

// ModalTitle renders a modal's heading. Every modal names itself, so a user
// who lands on one mid-flow can tell what it is about to do without having to
// infer it from the fields. It is the shared accent chip - appstyles.NormalTitle
// - stood off the body by its own margin, so a style or theme change to the
// chip lands on modals and panes alike.
//
// The margin replaces the blank line each caller would otherwise have to add -
// it matches the blank row the hint line sits above: the heading and the
// footer are the modal's chrome, and both stand off the body.
func ModalTitle(text string) string {
	return appstyles.NormalTitle().
		MarginBottom(1).
		Render(text)
}

// modalListChrome is the rows a list-in-a-modal spends on everything that is
// not a list row: ModalSurface's border (2) and padding (2), the ModalTitle
// and the blank row its margin leaves (2), the blank row above the hints (1),
// and the two hint lines (2).
const modalListChrome = 9

// ModalListHeight is the height to build a modal's list with so the modal
// fits a terminal termHeight rows tall.
//
// renderWithModal (src/model/View.go) centers a modal by clamping y to 0, so
// a modal taller than the terminal does not scroll or shrink - it loses its
// hint line and bottom border off the bottom of the screen. A list sized to
// len(items) is therefore a latent overflow on any project big enough, and
// the caller pairs this with SetShowPagination(height < len(items)) so the
// rows that no longer fit stay reachable and say so.
//
// The floor of 3 is deliberate: below about 12 rows there is no honest answer,
// and a terminal that short cannot show the modal's own chrome either.
func ModalListHeight(items, termHeight int) int {
	return min(items, max(3, termHeight-modalListChrome))
}

// ModalHints renders a modal's own help line, in the footer bar's format but
// with the lighter description color the modal surface needs. Every modal
// carries one: the footer bar is hidden behind the modal while it is open, so
// the keys the modal takes over are advertised here or nowhere.
func ModalHints(hints ...KeyHint) string {
	return RenderKeyHints(hints, appstyles.Active.TextMuted)
}

// EmptyCard renders a dim, centered, rounded-border card used for the
// empty / onboarding states. `key` is shown in the accent color inside
// brackets, `hint` is the trailing description in a dim color. `availHeight`
// is the vertical space in which the card should be centered. `bg` is the
// panel's background tier, which the space around the card sits on; the card
// itself is recessed below that tier so it reads as inset into the panel.
func EmptyCard(width, availHeight int, bg color.Color, title, body, key, hint string) string {
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

// PanelFrame renders the title chrome shared by DetailsPanel and
// GroupDetailsPanel, filling exactly the width x height box the panel was
// given. The 3-tier background system handles focus: tier 3 (panel) when
// unfocused, tier 4 (elevated) when focused.
//
// `titleRight` is an optional accessory pinned to the right end of the title
// row - the group status pill uses it. A blank row separates the title row
// from the body, so the panel's own label doesn't read as part of whatever
// the body starts with; PanelBodyHeight accounts for both rows.
//
// Callers embed their action buttons at the bottom of `body`, which pins the
// action row to the bottom of the panel.
// The chip itself is appstyles.NormalTitle; the MarginLeft(2) here is the
// frame's own left gutter, matching the 2 columns the bubbles list TitleBar
// adds inside the list wrappers - see appstyles.NormalTitle.
func PanelFrame(title string, titleRight string, isFocused bool, width int, height int, body string) string {
	bg := PanelBg(isFocused)

	style := FitBox(WrapperStyle.Background(bg), width, height)
	titleRow := appstyles.NormalTitle().MarginLeft(2).Render(title)

	if titleRight != "" {
		gap := max(0, PanelBodyWidth(width)-lipgloss.Width(titleRow)-lipgloss.Width(titleRight))

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

// PanelBodyWidth and PanelBodyHeight are the space a panel body gets inside a
// frame of the given total size: the frame's own padding taken off, plus the
// title row for the vertical axis. Callers size their content with these
// rather than with hardcoded offsets, so a change to WrapperStyle's padding
// doesn't silently push content out of the panel.
func PanelBodyWidth(total int) int {
	frameW, _ := WrapperStyle.GetFrameSize()

	return max(0, total-frameW)
}

func PanelBodyHeight(total int) int {
	_, frameH := WrapperStyle.GetFrameSize()

	return max(0, total-frameH-2) // -2 for the title row and the blank row under it
}
