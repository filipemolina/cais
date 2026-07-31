package components

import (
	"image/color"
	"slices"
	"strings"

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

// actionButton is one control in the row: the binding it stands for, and the
// two things the row needs to know that the binding does not carry.
type actionButton struct {
	binding key.Binding
	// drop is this button's place in the row's degradation order, lowest first.
	// It is separate from the slice order because the order to shed in is not
	// the order to read in: remove goes first because it is destructive and a
	// cramped click target is the last thing a destructive action should have,
	// then pull because it is the rarest, then logs; the three lifecycle verbs
	// are what a two-inch-wide panel keeps.
	drop int
	// danger marks the destructive one. It is a field rather than a comparison
	// against keys.Details.Remove at render time so that the row states its own
	// policy in one table instead of hiding it in a conditional.
	danger bool
}

// actionButtons are the bindings the row stands for, in the order it shows
// them - the same order the footer lists them in, so the eye can move between
// the two without re-sorting.
//
// The row is built from the bindings rather than from literal label/shortcut
// pairs so that it cannot claim a key the handlers don't answer to; that is the
// rule the footer and the help overlay already follow, and it is why logs is
// here at all. The row used to be a hand-written five that omitted `l logs`
// while the footer offered it, so the panel's own actions were advertised in
// two places that listed different things.
func actionButtons() []actionButton {
	return []actionButton{
		{binding: keys.Details.Start, drop: 5},
		{binding: keys.Details.Stop, drop: 4},
		{binding: keys.Details.Restart, drop: 3},
		{binding: keys.Details.Pull, drop: 1},
		{binding: keys.Details.Remove, drop: 0, danger: true},
		{binding: keys.Details.Logs, drop: 2},
	}
}

// actionButtonGap is the column between two chips. The chips carry their own
// fill, so the gap is what makes them read as separate controls.
const actionButtonGap = 1

// renderActionButtons renders the action row shared by DetailsPanel and
// GroupDetailsPanel, right-aligned within the panel body width it is given.
// `bg` is the panel's background tier, which the row sits on and the chips are
// recessed into.
//
// `ctx` is the same screen state the footer reports, and it decides which
// buttons are live. Before it was threaded through, the row painted every
// button identically whatever the screen was doing, which read as a promise
// that s/t/r/p/x/l work at all times - they only work while the details panel
// holds focus. Asking keys.Live is what makes the row and the footer one
// statement rather than two that happen to agree.
//
// A row too wide for the panel sheds buttons rather than wrapping. lipgloss
// wraps on the cell, not on the control, so an overflowing row used to break a
// chip across two lines and push the panel's own content out of its box; every
// shed key is still on the footer and still pressable, which a mangled row is
// not. See the narrow-terminal entry in TODO.md.
func renderActionButtons(width int, bg color.Color, ctx keys.Context) string {
	shown := actionButtons()

	row := joinActionButtons(shown, bg, ctx)
	for len(shown) > 0 && lipgloss.Width(row) > width {
		shown = slices.DeleteFunc(shown, func(b actionButton) bool {
			return b.drop == lowestDrop(shown)
		})
		row = joinActionButtons(shown, bg, ctx)
	}

	// The chips are opaque and the gaps are painted, so there is nothing left
	// unstyled inside the row; the seal is kept for the alignment padding the
	// Width() below adds around it.
	row = appstyles.FillBackground(bg, row)

	return lipgloss.NewStyle().
		Width(max(0, width)).
		AlignHorizontal(lipgloss.Right).
		Background(bg).
		Render(row)
}

// joinActionButtons renders buttons into a single row on bg, each chip's
// enabled state resolved against ctx.
func joinActionButtons(buttons []actionButton, bg color.Color, ctx keys.Context) string {
	if len(buttons) == 0 {
		return ""
	}

	gap := lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", actionButtonGap))

	parts := make([]string, 0, len(buttons)*2-1)
	for i, button := range buttons {
		if i > 0 {
			parts = append(parts, gap)
		}

		help := button.binding.Help()
		parts = append(parts, Button(ButtonSpec{
			Text:     buttonLabel(help.Desc),
			Shortcut: help.Key,
			Enabled:  keys.Live(ctx, button.binding),
			Danger:   button.danger,
		}).View().Content)
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

// lowestDrop is the drop rank of the next button to shed.
func lowestDrop(buttons []actionButton) int {
	lowest := buttons[0].drop
	for _, button := range buttons[1:] {
		lowest = min(lowest, button.drop)
	}

	return lowest
}

// buttonLabel is a binding's help description as a button label: the footer
// prints "start" mid-sentence, a button is captioned "Start". Capitalizing the
// footer's word is what lets both come from the one binding.
func buttonLabel(desc string) string {
	if desc == "" {
		return desc
	}

	return strings.ToUpper(desc[:1]) + desc[1:]
}

// panelRule is the thin horizontal line both details panels separate their
// sections with. One helper rather than five copies of the same three-line
// style, so the panels cannot drift to different rules.
func panelRule(width int) string {
	return lipgloss.NewStyle().
		Foreground(appstyles.Active.BorderDefault).
		Width(width).
		Render(strings.Repeat("─", max(width, 0)))
}

// panelBodyWithActions lays out a details panel's body: `content` at the top,
// `footer` on the body's last rows, and blank rows between them. Both details
// panels build their body through here, which is what makes the action row land
// on the same line of both rather than wherever each panel's content happens to
// end - see "The action row" in docs/DESIGN.md.
//
// The content is clipped before the footer is attached, so a panel whose
// content outgrows its box loses its last rows rather than its actions. Doing
// it the other way round - joining first and clipping the result - takes the
// bottom off, which is exactly the row that has to survive.
func panelBodyWithActions(width, avail int, bg color.Color, content, footer string) string {
	contentAvail := max(0, avail-lipgloss.Height(footer))

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
	parts = append(parts, footer)

	body := lipgloss.JoinVertical(lipgloss.Left, parts...)

	// Safety cap: a miscount must never grow the panel past its body region.
	if avail > 0 {
		body = lipgloss.NewStyle().MaxHeight(avail).Render(body)
	}

	return body
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
// infer it from the fields. It is the shared accent chip - appstyles.NormalTitle
// - stood off the body by its own margin, so a style or theme change to the
// chip lands on modals and panes alike.
//
// The margin replaces the blank line each caller would otherwise have to add -
// it matches the blank row the hint line sits above: the heading and the
// footer are the modal's chrome, and both stand off the body.
func modalTitle(text string) string {
	return appstyles.NormalTitle().
		MarginBottom(1).
		Render(text)
}

// modalListChrome is the rows a list-in-a-modal spends on everything that is
// not a list row: modalSurface's border (2) and padding (2), the modalTitle
// and the blank row its margin leaves (2), the blank row above the hints (1),
// and the two hint lines (2).
const modalListChrome = 9

// modalListHeight is the height to build a modal's list with so the modal
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
func modalListHeight(items, termHeight int) int {
	return min(items, max(3, termHeight-modalListChrome))
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
// The chip itself is appstyles.NormalTitle; the MarginLeft(2) here is the
// frame's own left gutter, matching the 2 columns the bubbles list TitleBar
// adds inside the list wrappers - see appstyles.NormalTitle.
func renderPanelFrame(title string, titleRight string, isFocused bool, width int, height int, body string) string {
	bg := panelBg(isFocused)

	style := fitBox(wrapperStyle.Background(bg), width, height)
	titleRow := appstyles.NormalTitle().MarginLeft(2).Render(title)

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
