package containerslist

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

type containersListCustomDelegate struct {
	isParentFocused bool
}

func (d containersListCustomDelegate) Height() int                             { return 4 }
func (d containersListCustomDelegate) Spacing() int                            { return 0 }
func (d containersListCustomDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

// Render handles the actual drawing of the item
func (d containersListCustomDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	// Cast the generic list.Item back to our specific ContainerListItem
	item, ok := listItem.(apptypes.ContainerListItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()
	isActive := false

	// The row's left edge is the same solid bar the nav uses for its active
	// tab ("▌"), so list rows and the nav agree on thickness. State is carried
	// by color alone: accent = cursor row, primary = selected, muted = default.
	barColor := appstyles.Active.TextMuted

	if isSelected && d.isParentFocused {
		barColor = appstyles.Active.Accent
	} else if isActive {
		barColor = appstyles.Active.TextPrimary
	}

	wrapperStyle := lipgloss.NewStyle().
		Width(m.Width() - 1).
		Padding(1)

	// The selected row lifts to the modal surface; the bar must share that
	// background so the row renders as one solid strip, as the old border did.
	if isSelected && d.isParentFocused {
		wrapperStyle = wrapperStyle.Background(appstyles.Active.ModalBg)
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Width(m.Width() - 1)

	var barBg color.Color
	if isSelected && d.isParentFocused {
		barBg = appstyles.Active.ModalBg
	}

	title := titleStyle.Render(item.Title())
	description := item.Description(isSelected && d.isParentFocused)

	content := wrapperStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, description))

	// The bar spans the row's full height, one ▌ per line, rather than a sliver
	// at the top - the nav's single-line bar stretched to the row's height.
	bar := chrome.BarColumn(barColor, barBg, content)

	// Print the styled string to the Bubble Tea io.Writer
	fmt.Fprint(w, lipgloss.JoinHorizontal(lipgloss.Left, bar, content))
}

func (m Model) View() tea.View {
	wrapper := lipgloss.NewStyle().
		Padding(1, 2, 2, 2)

	if m.isFocused {
		wrapper = wrapper.
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(appstyles.Active.Accent).
			Padding(0, 1, 1, 1)
	}

	// The title chip is restyled here, on a copy, rather than in the
	// constructor - see appstyles.NormalTitle for why.
	l := m.list
	l.Styles.Title = appstyles.NormalTitle()

	renderedList := wrapper.Render(l.View())

	v := tea.NewView(renderedList)
	return v
}
