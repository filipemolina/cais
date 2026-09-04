package backupslist

import (
	"fmt"
	"image/color"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/utils"
)

// sourceColWidth caps the source tag shown on each row, e.g. "compose" or
// ".env", so the columns line up.
const sourceColWidth = 8

func (m Model) View() tea.View {
	bg := chrome.PanelBg()

	bodyWidth := max(1, chrome.PanelBodyWidth(m.panelWidth))
	bodyAvail := max(1, chrome.PanelBodyHeight(m.panelHeight))

	var body string
	switch {
	case m.loading:
		body = chrome.EmptyCard(bodyWidth, bodyAvail, bg,
			"Loading backups",
			"Reading the backup store…",
			"", "")

	case m.loadErr != nil:
		body = chrome.EmptyCard(bodyWidth, bodyAvail, bg,
			"Could not read backups",
			m.loadErr.Error(),
			"", "")

	case m.empty:
		body = chrome.EmptyCard(bodyWidth, bodyAvail, bg,
			"No backups yet",
			"Edits to this stack are saved automatically. Once a compose or .env write lands, its prior copies appear here.",
			"", "")

	default:
		body = m.renderList(bodyWidth, bodyAvail, bg)
	}

	titleRight := ""
	if len(m.entries) > 0 {
		titleRight = lipgloss.NewStyle().
			Foreground(appstyles.Active.TextDim).
			Render(fmt.Sprintf("%d version%s", len(m.entries), plural(len(m.entries))))
	}

	screen := chrome.PanelFrame("Backups", titleRight, m.panelWidth, m.panelHeight, body)
	return tea.NewView(screen)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// renderList renders the rows inside the scroll window.
//
// There is no column header: a row is a filename over a timestamp, stacked,
// not two columns, so there is nothing for a two-column header to label. The
// groups and services lists carry no header either, which is the language
// this list follows.
func (m Model) renderList(width int, avail int, bg color.Color) string {
	if width < 1 {
		width = 1
	}

	var parts []string

	// Only the rows inside the window are rendered. Rendering the whole
	// history and letting the body clip is what this phase replaced, and
	// rendering it into a viewport costs a re-render of every row on every
	// cursor move - see the note on ensureCursorVisible.
	end := min(len(m.entries), m.rowOffset+m.visibleRows())
	for i := m.rowOffset; i < end; i++ {
		parts = append(parts, m.renderRow(i, m.entries[i], width))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)

	return chrome.PanelBodyWithFooter(width, avail, bg, content, "")
}

// rowContentWidth is the room a row's text actually gets: the panel body
// less the bar column and the wrapper's own horizontal padding.
func rowContentWidth(width int) int {
	return max(0, (width-1)-2)
}

// renderRow draws one stored version the way the groups and services lists
// draw their rows, because it is the same kind of thing on the same kind of
// page and should read as one visual language: a solid bar down the left
// edge carrying state by colour alone - accent for the cursor row, muted
// grey otherwise - over content set in a wrapper with a row of padding on
// every side.
//
// The first line is the live file's own name rather than the "compose" /
// ".env" tag behind it, because compose.yml and compose.yaml are different
// files and a list that calls both "compose" cannot say which one a copy
// would be restored over. The sha is not here: it identifies content, which
// is what the preview panel beside this list is showing.
func (m Model) renderRow(idx int, entry utils.BackupEntry, width int) string {
	isSelected := idx == m.selectedIdx
	rowBg := chrome.ListRowBg(isSelected)

	// The row's left edge is the same solid bar the nav uses for its active
	// tab ("▌"), so list rows and the nav agree on thickness. This list has
	// one cursor and no second "selected" state, so it uses two of the three
	// colours the other lists carry.
	barColor := appstyles.Active.TextMuted
	if isSelected {
		barColor = appstyles.Active.Accent
	}

	wrapperStyle := lipgloss.NewStyle().
		Width(width - 1).
		Padding(1).
		Background(rowBg)

	// The wrapper's Width includes its Padding(1), so the content area is two
	// columns narrower than the wrapper.
	contentWidth := rowContentWidth(width)

	// The filename is this row's title - the thing that names what would be
	// restored - so it carries the cursor's bold, the way a service name does.
	file := lipgloss.NewStyle().
		Bold(isSelected).
		Foreground(appstyles.Active.TextPrimary).
		Background(rowBg).
		Width(contentWidth).
		Render(chrome.Truncate(entry.File, contentWidth))

	// UTC is spelled out on the row now that there is no column header to
	// say so, because a bare timestamp invites being read as local time.
	when := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextDim).
		Background(rowBg).
		Width(contentWidth).
		Render(chrome.Truncate(
			entry.Timestamp.UTC().Format("2006-01-02 15:04")+" UTC",
			contentWidth,
		))

	content := wrapperStyle.Render(lipgloss.JoinVertical(lipgloss.Left, file, when))

	// The bar spans the row's full height, one ▌ per line, rather than a
	// sliver at the top - the nav's single-line bar stretched to the row's
	// height.
	bar := chrome.BarColumn(barColor, rowBg, content)

	// Seal the row against its own background: JoinVertical pads the shorter
	// line out with unstyled spaces, which would otherwise show the terminal
	// background through the row.
	return appstyles.FillBackground(rowBg, lipgloss.JoinHorizontal(lipgloss.Left, bar, content))
}
