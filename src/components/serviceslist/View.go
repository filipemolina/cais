package serviceslist

import (
	"fmt"
	"image/color"
	"io"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
)

/*
 * Styling by creating a custom delegate
 */

type servicesListCustomDelegate struct {
	isParentFocused bool
	activeIndex     int
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

	rowBg := chrome.ListRowBg(isActive, d.isParentFocused)

	// The row's left edge is the same solid bar the nav uses for its active
	// tab ("▌"), so list rows and the nav agree on thickness. State is carried
	// by color alone: accent = cursor row, primary = selected, muted = default.
	barColor := appstyles.Active.TextMuted

	if isActive {
		barColor = appstyles.Active.Accent
	} else if isSelected && d.isParentFocused {
		barColor = appstyles.Active.TextPrimary
	}

	wrapperStyle := lipgloss.NewStyle().
		Width(m.Width() - 1).
		Padding(1).
		Background(rowBg)

	// The title style only bolds the active row; the selected row's bold comes
	// from the wrapper, which is why it is applied here rather than in the title.
	if isSelected && d.isParentFocused && !isActive {
		wrapperStyle = wrapperStyle.Bold(true)
	}

	titleStyle := lipgloss.NewStyle().
		Bold(isActive).
		Foreground(titleColor).
		Background(rowBg).
		Width(m.Width() - 1)

	title := titleStyle.Render(item.Title())
	description := item.Description(isActive)

	content := wrapperStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, description))

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
	bg := chrome.PanelBg(m.isFocused)

	wrapper := chrome.FitBox(chrome.ListWrapperStyle.Background(bg), m.panelWidth, m.panelHeight)

	// The title chip is restyled here, on a copy, rather than in the
	// constructor - see appstyles.NormalTitle for why.
	l := m.list
	l.Styles.Title = appstyles.NormalTitle()

	// The list joins its title, rows and paginator internally, padding the
	// short ones with unstyled spaces; seal them against the panel tier. Rows
	// arrive already sealed against their own background, so this only fills
	// what the list itself left bare.
	v := tea.NewView(wrapper.Render(appstyles.FillBackground(bg, l.View())))
	return v
}
