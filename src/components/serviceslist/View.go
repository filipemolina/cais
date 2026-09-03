package serviceslist

import (
	"fmt"
	"image/color"
	"io"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/components/chrome"
)

/*
 * Styling by creating a custom delegate
 */

type servicesListCustomDelegate struct {
	activeIndex int
}

func (d servicesListCustomDelegate) Height() int                             { return 4 }
func (d servicesListCustomDelegate) Spacing() int                            { return 0 }
func (d servicesListCustomDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

// Render handles the actual drawing of the item
func (d servicesListCustomDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	// Cast the generic list.Item back to our specific ServiceListItem
	item, ok := listItem.(apptypes.ServiceListItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()
	isActive := index == d.activeIndex
	var titleColor color.Color

	if isActive {
		titleColor = appstyles.Active.TextPrimary
	} else {
		titleColor = appstyles.Active.TextMuted
	}

	rowBg := chrome.ListRowBg(isActive)

	// The row's left edge is the same solid bar the nav uses for its active
	// tab ("▌"), so list rows and the nav agree on thickness. State is carried
	// by color alone: accent = cursor row, primary = selected, muted = default.
	barColor := appstyles.Active.TextMuted

	if isActive {
		barColor = appstyles.Active.Accent
	} else if isSelected {
		barColor = appstyles.Active.TextPrimary
	}

	wrapperStyle := lipgloss.NewStyle().
		Width(m.Width() - 1).
		Padding(1).
		Background(rowBg)

	// The title style only bolds the active row; the selected row's bold comes
	// from the wrapper, which is why it is applied here rather than in the title.
	if isSelected && !isActive {
		wrapperStyle = wrapperStyle.Bold(true)
	}

	// The status dot rides the far right of the title line, the same place
	// the groups list puts its group dot, so the two panels read as one.
	//
	// It is always drawn, which is where it parts company with the groups
	// list: a group has a third, partly-running state and hides its dot when
	// nothing in it runs, but a service either runs or it does not and a
	// missing dot would be indistinguishable from a missing answer.
	dot := statusDot(item, rowBg)

	// The wrapper's Width includes its Padding(1), so the content area is two
	// columns narrower than the wrapper. The title and the dot share that
	// content area on one line: the title yields the dot's column and is
	// truncated with an ellipsis when it does not fit, so the dot always
	// shows even at the cost of a shortened name.
	contentWidth := max(0, (m.Width()-1)-2)
	titleWidth := max(0, contentWidth-lipgloss.Width(dot))

	titleStyle := lipgloss.NewStyle().
		Bold(isActive).
		Foreground(titleColor).
		Background(rowBg).
		Width(titleWidth)

	title := titleStyle.Render(chrome.Truncate(item.Title(), titleWidth))

	titleRow := lipgloss.JoinHorizontal(lipgloss.Left, title, dot)
	description := item.Description(isActive)

	content := wrapperStyle.Render(lipgloss.JoinVertical(lipgloss.Left, titleRow, description))

	// The bar spans the row's full height, one ▌ per line, rather than a sliver
	// at the top - the nav's single-line bar stretched to the row's height.
	bar := chrome.BarColumn(barColor, rowBg, content)

	// Seal the row against its own background before handing it to the list:
	// JoinVertical pads the description out to the title's width with unstyled
	// spaces, which would otherwise show the terminal background through the
	// row. Sealing here (rather than over the whole list) is what keeps the
	// active row's lighter surface color from being flattened to the panel's.
	row := appstyles.FillBackground(rowBg, lipgloss.JoinHorizontal(lipgloss.Left, bar, content))

	// Print the styled string to the Bubble Tea io.Writer
	fmt.Fprint(w, row)
}

func (m Model) View() tea.View {
	// Same 3-tier treatment as the groups list: focus lifts the panel from
	// tier 3 to tier 4 rather than adding a border, so the panel's box stays
	// the same size whether or not it is focused.
	bg := chrome.PanelBg()

	wrapper := chrome.FitBox(chrome.ListWrapperStyle.Background(bg), m.panelWidth, m.panelHeight)

	// The title chip is restyled here, on a copy, rather than in the
	// constructor - see appstyles.NormalTitle for why.
	l := m.list
	l.Styles.Title = appstyles.NormalTitle()
	// Only ever on screen while a filter is standing - see Update.
	l.Styles.StatusBar = appstyles.FilterStatus()
	l.Styles.StatusBarFilterCount = lipgloss.NewStyle().Foreground(appstyles.Active.TextMuted)
	l.Styles.DividerDot = lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).SetString(" • ")

	// The list joins its title, rows and paginator internally, padding the
	// short ones with unstyled spaces; seal them against the panel tier. Rows
	// arrive already sealed against their own background, so this only fills
	// what the list itself left bare.
	v := tea.NewView(wrapper.Render(appstyles.FillBackground(bg, l.View())))
	return v
}

// statusDot returns the styled status glyph for a service row: a solid circle,
// StatusRunning when the service's container is running and StatusError when
// it is not.
//
// StatusError rather than StatusStopped, which is what a dot elsewhere in the
// app uses, because at the size of a single glyph the grey did not carry.
// StatusStopped is a desaturated blue-grey a shade off the muted body text, so
// against StatusRunning's green the pair read as "coloured" and "not
// coloured" rather than as two states - and on a list where most rows are one
// or the other, that is the whole job. StatusError is the fill the STOPPED
// pill already uses (detailspanel and groupdetailspanel), so the row and the
// panel it opens agree on what stopped looks like. Both are theme tokens and
// follow the active theme.
//
// "Not running" folds together a stopped container and no container at all,
// which is the reading ServiceListItem.StatusPill and the details panel's own
// status line already take: from the row's point of view a service with
// nothing behind it is not running. The dot is rendered on the row background
// so it stays legible across selection and focus states.
func statusDot(item apptypes.ServiceListItem, rowBg color.Color) string {
	dotColor := appstyles.Active.StatusError
	if item.Status == "running" {
		dotColor = appstyles.Active.StatusRunning
	}

	return lipgloss.NewStyle().
		Foreground(dotColor).
		Background(rowBg).
		Render("●")
}
