package backuppage

import (
	"fmt"
	"image/color"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/utils"
)

// sourceLabel renders the short source tag shown on each row, e.g. "compose"
// or ".env", capped to a fixed width so the columns line up.
const sourceColWidth = 8

func (m Model) View() tea.View {
	// Always the focused tier: see the note on the model.
	bg := chrome.PanelBg(true)

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
		body = m.renderSplit(bodyWidth, bodyAvail, bg)
	}

	titleRight := ""
	if len(m.entries) > 0 {
		titleRight = lipgloss.NewStyle().
			Foreground(appstyles.Active.TextDim).
			Render(fmt.Sprintf("%d version%s", len(m.entries), plural(len(m.entries))))
	}

	screen := chrome.PanelFrame("Backups", titleRight, true, m.panelWidth, m.panelHeight, body)
	return tea.NewView(screen)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// renderSplit lays the list and the preview side by side, filling the panel
// body the way a two-column panel does. The list is a fixed-width column on
// the left; the preview viewport fills the rest.
func (m Model) renderSplit(bodyWidth, bodyAvail int, bg color.Color) string {
	listCol := m.renderList(m.listWidth, bodyAvail, bg)
	previewCol := m.renderPreview(m.previewWidth, bodyAvail, bg)

	return lipgloss.JoinHorizontal(lipgloss.Top, listCol, previewCol)
}

// renderList renders the version rows: source, UTC timestamp, and SHA-8. The
// selected row is lifted to the surface tier with an accent bar down its left
// edge, the same selection language the env/file rows use.
func (m Model) renderList(width int, avail int, bg color.Color) string {
	if width < 1 {
		width = 1
	}

	parts := []string{
		m.renderListHeader(width),
		chrome.PanelRule(width),
	}

	for i, entry := range m.entries {
		parts = append(parts, m.renderRow(i, entry, width))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return chrome.PanelBodyWithFooter(width, avail, bg, content, "")
}

func (m Model) renderListHeader(width int) string {
	dim := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).Bold(true)

	src := dim.Width(sourceColWidth).Render("SOURCE")
	ts := dim.Width(width - sourceColWidth - 1).Render("UTC TIME")

	return lipgloss.JoinHorizontal(lipgloss.Left,
		lipgloss.NewStyle().Width(1).Render(""),
		src,
		ts,
	)
}

func (m Model) renderRow(idx int, entry utils.BackupEntry, width int) string {
	isSelected := idx == m.selectedIdx
	rowBg := chrome.ListRowBg(isSelected, true)

	ts := entry.Timestamp.UTC().Format("2006-01-02 15:04")
	src := entry.Source

	srcStyle := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextMuted).
		Background(rowBg).
		Width(sourceColWidth).
		Render(chrome.Truncate(src, sourceColWidth))

	tsStyle := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextPrimary).
		Background(rowBg).
		Render(chrome.Truncate(ts, width-sourceColWidth-1))

	content := lipgloss.JoinHorizontal(lipgloss.Left, srcStyle, tsStyle)

	shaHint := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextDim).
		Background(rowBg).
		Width(width - 1).
		Render("sha8 " + entry.SHA8)

	row := lipgloss.JoinVertical(lipgloss.Left, content, shaHint)

	barColor := rowBg
	if isSelected {
		barColor = appstyles.Active.Accent
	}
	bar := chrome.BarColumn(barColor, rowBg, row)

	full := lipgloss.JoinHorizontal(lipgloss.Left, bar, row)
	return appstyles.FillBackground(rowBg, full)
}

// renderPreview renders the selected copy's content in a viewport, with a
// short header naming the source and SHA-8.
func (m Model) renderPreview(width int, avail int, bg color.Color) string {
	if width < 1 {
		width = 1
	}

	header := ""
	if sel := m.selectedIdx; sel >= 0 && sel < len(m.entries) {
		e := m.entries[sel]
		header = lipgloss.NewStyle().
			Foreground(appstyles.Active.TextDim).
			Background(bg).
			Width(width).
			Render(fmt.Sprintf("preview · %s · %s", e.Source, e.SHA8))
	}

	vp := m.previewVP.View()
	vp = appstyles.FillBackground(bg, vp)

	// Clip the viewport to the available height below the header row.
	availForVP := max(0, avail-1)
	vp = lipgloss.NewStyle().MaxHeight(availForVP).Render(vp)

	return lipgloss.JoinVertical(lipgloss.Left, header, vp)
}
