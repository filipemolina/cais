package chrome

import (
	"charm.land/lipgloss/v2"

	"github.com/filipemolina/cais/src/appstyles"
)

// ListRowInput is the per-row material a body-list delegate supplies. Only
// the parts that genuinely differ between the two lists are here; everything
// else about a row - its bar, its surface, how the title yields to the dot -
// is ListRow's business.
type ListRowInput struct {
	// Title is the row's label, truncated to whatever the dot leaves it.
	Title string

	// Dot is the pre-styled status glyph pinned to the right of the title
	// row, or "" for no dot. The groups list omits it for a stopped group;
	// the services list always draws one.
	Dot string

	// Body is any further lines drawn under the title row, already styled.
	// The services list passes its description; the groups list passes
	// nothing. A delegate's Height() must match the lines actually emitted
	// here - see GroupsListCustomDelegate.Height.
	Body []string

	// Width is the list's width, not the row's: ListRow subtracts the bar
	// column itself so the two lists cannot disagree about it.
	Width int

	// IsSelected marks the row under the cursor; IsActive marks the row the
	// panel is acting on. They differ while the cursor moves in an unfocused
	// list.
	IsSelected bool
	IsActive   bool
}

// ListRow renders one row of a body list: a full-height color bar, then a
// title row on the row's own surface with an optional status dot pinned right,
// then any body lines.
//
// The services and groups delegates drew this identically for about fifty
// lines each, down to the comments. They differ only in the item type they
// cast to, the dot they compute, and whether a description follows the title -
// so those are what ListRowInput carries and nothing else.
func ListRow(in ListRowInput) string {
	rowBg := ListRowBg(in.IsActive)

	titleColor := appstyles.Active.TextMuted
	if in.IsActive {
		titleColor = appstyles.Active.TextPrimary
	}

	// The row's left edge is the same solid bar the nav uses for its active
	// tab ("▌"), so list rows and the nav agree on thickness. State is carried
	// by color alone: accent = cursor row, primary = selected, muted = default.
	barColor := appstyles.Active.TextMuted
	if in.IsActive {
		barColor = appstyles.Active.Accent
	} else if in.IsSelected {
		barColor = appstyles.Active.TextPrimary
	}

	wrapperStyle := lipgloss.NewStyle().
		Width(in.Width - 1).
		Padding(1).
		Background(rowBg)

	// The title style only bolds the active row; the selected row's bold comes
	// from the wrapper, which is why it is applied here rather than in the title.
	if in.IsSelected && !in.IsActive {
		wrapperStyle = wrapperStyle.Bold(true)
	}

	// The wrapper's Width includes its Padding(1), so the content area is two
	// columns narrower than the wrapper. The title and the dot share that
	// content area on one line: the title yields the dot's column and is
	// truncated with an ellipsis when it does not fit, so the dot always shows
	// even at the cost of a shortened name.
	contentWidth := max(0, (in.Width-1)-2)
	titleWidth := max(0, contentWidth-lipgloss.Width(in.Dot))

	titleStyle := lipgloss.NewStyle().
		Bold(in.IsActive).
		Foreground(titleColor).
		Background(rowBg).
		Width(titleWidth)

	title := titleStyle.Render(Truncate(in.Title, titleWidth))
	titleRow := lipgloss.JoinHorizontal(lipgloss.Left, title, in.Dot)

	content := wrapperStyle.Render(lipgloss.JoinVertical(lipgloss.Left, append([]string{titleRow}, in.Body...)...))

	// The bar spans the row's full height, one ▌ per line, rather than a sliver
	// at the top - the nav's single-line bar stretched to the row's height.
	bar := BarColumn(barColor, rowBg, content)

	// Seal the row against its own background before handing it to the list:
	// JoinVertical pads shorter body lines out to the title's width with
	// unstyled spaces, which would otherwise show the terminal background
	// through the row. Sealing here (rather than over the whole list) is what
	// keeps the active row's lighter surface color from being flattened to the
	// panel's.
	return appstyles.FillBackground(rowBg, lipgloss.JoinHorizontal(lipgloss.Left, bar, content))
}
