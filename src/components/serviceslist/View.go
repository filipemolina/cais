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

	isActive := index == d.activeIndex

	// The dot and the description are styled on the row's surface, so they
	// need the same background ListRow will use - ListRowBg is the shared
	// answer rather than a second guess at it.
	rowBg := chrome.ListRowBg(isActive)

	// The status dot is always drawn, which is where this parts company with
	// the groups list: a group has a third, partly-running state and hides its
	// dot when nothing in it runs, but a service either runs or it does not
	// and a missing dot would be indistinguishable from a missing answer.
	fmt.Fprint(w, chrome.ListRow(chrome.ListRowInput{
		Title:      item.Title(),
		Dot:        statusDot(item, rowBg),
		Body:       []string{item.Description(isActive)},
		Width:      m.Width(),
		IsSelected: index == m.Index(),
		IsActive:   isActive,
	}))
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
// StatusRunning when the service's container is running, StatusError when it
// is not, and dim while the answer is still unknown.
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
// "Not running" folds together a stopped container and no container at all -
// from the row's point of view a service with nothing behind it is not
// running - but only once docker has actually been asked; Model.containerStatus
// is what keeps that distinct from the third state. The dot is rendered on the
// row background so it stays legible across selection and focus states.
func statusDot(item apptypes.ServiceListItem, rowBg color.Color) string {
	return chrome.StateDot(item.Status, rowBg)
}
