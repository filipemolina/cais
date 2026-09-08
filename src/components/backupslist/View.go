package backupslist

import (
	"fmt"
	"image/color"
	"io"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/components/chrome"
)

// timestampLayout is how a stored copy's write time is spelled, here and in
// the filter value. UTC is spelled out beside it on the row: there is no
// column header to say so, and a bare timestamp invites being read as local
// time.
const timestampLayout = "2006-01-02 15:04"

/*
 * Styling by creating a custom delegate
 */

// backupsListCustomDelegate draws the version rows.
//
// It carries the panel's focus rather than an activeIndex, which is where it
// parts company with the groups and services delegates. Those lists have two
// states - the cursor row and the selected row, which can differ - because
// their selection is set from outside. Here the cursor is the selection:
// moving it publishes the copy, so there is no second row to mark. What the
// panel does have that they do not is a focus tier, and the rows sit flush on
// it, so the flag has to reach the delegate. See Model.setFocus.
type backupsListCustomDelegate struct {
	panelFocused bool
}

func (d backupsListCustomDelegate) Height() int                             { return 4 }
func (d backupsListCustomDelegate) Spacing() int                            { return 0 }
func (d backupsListCustomDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

// Render draws one stored version the way the groups and services lists draw
// their rows, because it is the same kind of thing on the same kind of page
// and should read as one visual language: a solid bar down the left edge
// carrying state by colour alone - accent for the cursor row, muted grey
// otherwise - over content set in a wrapper with a row of padding on every
// side.
//
// The sha is not on the row: it identifies content, which is what the preview
// panel beside this list is showing. It is still filterable - see
// backupItem.FilterValue.
func (d backupsListCustomDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(backupItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	// The rows take the panel's own tier as a parameter rather than picking a
	// tint of their own, so they stay flush when focus lifts the panel.
	rowBg := chrome.ListRowBgOn(isSelected, chrome.PanelBgFor(d.panelFocused))

	// The row's left edge is the same solid bar the nav uses for its active
	// tab ("▌"), so list rows and the nav agree on thickness. This list has
	// one cursor and no second "selected" state, so it uses two of the three
	// colours the other lists carry.
	barColor := appstyles.Active.TextMuted
	if isSelected {
		barColor = appstyles.Active.Accent
	}

	wrapperStyle := lipgloss.NewStyle().
		Width(m.Width() - 1).
		Padding(1).
		Background(rowBg)

	// The wrapper's Width includes its Padding(1), so the content area is two
	// columns narrower than the wrapper.
	contentWidth := max(0, (m.Width()-1)-2)

	// The filename is this row's title - the thing that names what would be
	// restored - so it carries the cursor's bold, the way a service name does.
	title := item.Title()

	// The copy that matches the live file right now says so, on the title's
	// own line: the marker qualifies which file the row is a copy of, and
	// the timestamp line is about when, not what. It is dropped whole when
	// the two do not fit rather than truncated, because a partial
	// "(curr…" is noise, and the title is the row's identity - the
	// marker is the first thing to go.
	marker := ""
	if item.isCurrent {
		marker = " (current)"
		if lipgloss.Width(title)+lipgloss.Width(marker) > contentWidth {
			marker = ""
		}
	}

	titleStyle := lipgloss.NewStyle().
		Bold(isSelected).
		Foreground(appstyles.Active.TextPrimary).
		Background(rowBg)

	var file string
	if marker == "" {
		file = titleStyle.
			Width(contentWidth).
			Render(chrome.Truncate(title, contentWidth))
	} else {
		// The marker runs one emphasis step below the title and never takes
		// the accent: the bar carries cursor state, this carries content
		// state, and the two are different channels.
		markerFg := appstyles.Active.TextMuted
		if isSelected {
			markerFg = appstyles.Active.TextPrimary
		}
		file = lipgloss.NewStyle().
			Width(contentWidth).
			Background(rowBg).
			Render(lipgloss.JoinHorizontal(lipgloss.Left,
				titleStyle.Render(title),
				lipgloss.NewStyle().Foreground(markerFg).Background(rowBg).Render(marker),
			))
	}

	when := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextDim).
		Background(rowBg).
		Width(contentWidth).
		Render(chrome.Truncate(
			item.entry.Timestamp.UTC().Format(timestampLayout)+" UTC",
			contentWidth,
		))

	content := wrapperStyle.Render(lipgloss.JoinVertical(lipgloss.Left, file, when))

	// The bar spans the row's full height, one ▌ per line, rather than a
	// sliver at the top - the nav's single-line bar stretched to the row's
	// height.
	bar := chrome.BarColumn(barColor, rowBg, content)

	// Seal the row against its own background before handing it to the list:
	// JoinVertical pads the shorter line out with unstyled spaces, which would
	// otherwise show the terminal background through the row.
	fmt.Fprint(w, appstyles.FillBackground(rowBg, lipgloss.JoinHorizontal(lipgloss.Left, bar, content)))
}

func (m Model) View() tea.View {
	// The panel's whole tier moves with focus, frame included, rather than a
	// heavier border appearing - see chrome.PanelBgFor. This is the one body
	// list with a focus stop; the groups and services lists call PanelBg.
	bg := chrome.PanelBgFor(m.focused)

	wrapper := chrome.FitBox(chrome.ListWrapperStyle.Background(bg), m.panelWidth, m.panelHeight)

	frameW, frameH := chrome.ListWrapperStyle.GetFrameSize()
	contentWidth := max(0, m.panelWidth-frameW)
	contentHeight := max(0, m.panelHeight-frameH)

	var content string

	switch {
	case m.loading:
		content = m.stateView(bg, contentWidth, contentHeight,
			"Loading backups",
			"Reading the backup store…")

	case m.loadErr != nil:
		content = m.stateView(bg, contentWidth, contentHeight,
			"Could not read backups",
			m.loadErr.Error())

	case m.ListEmpty():
		content = m.stateView(bg, contentWidth, contentHeight,
			"No backups yet",
			"Edits to this stack are saved automatically. Once a compose or .env write lands, its prior copies appear here.")

	default:
		// The title chip is restyled here, on a copy, rather than in the
		// constructor - see appstyles.NormalTitle for why.
		l := m.list
		l.Styles.Title = appstyles.NormalTitle()
		l.Styles.StatusBar = appstyles.FilterStatus()
		l.Styles.StatusBarFilterCount = lipgloss.NewStyle().Foreground(appstyles.Active.TextMuted)
		l.Styles.DividerDot = lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).SetString(" • ")

		// The list joins its title, rows and paginator internally, padding the
		// short ones with unstyled spaces; seal them against the panel tier.
		// Rows arrive already sealed against their own background.
		content = appstyles.FillBackground(bg, l.View())
	}

	return tea.NewView(wrapper.Render(content))
}

// stateView is the panel with no rows to show: the title chip over a message.
// The chip is rendered by hand because the bubbles list only draws its own
// while it has rows, and a panel that loses its label when the store is empty
// reads as a panel that failed to load rather than as an empty one. Same
// approach as the groups list's empty state.
func (m Model) stateView(bg color.Color, width, height int, title, body string) string {
	titleRow := appstyles.NormalTitle().MarginLeft(2).Render(m.list.Title)

	bodyStyle := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextMuted).
		Background(bg).
		Padding(2, 2)

	rendered := chrome.FitBox(bodyStyle, width, max(0, height-1)).Render(title + "\n\n" + body)

	return appstyles.FillBackground(bg, lipgloss.JoinVertical(lipgloss.Left, titleRow, rendered))
}
