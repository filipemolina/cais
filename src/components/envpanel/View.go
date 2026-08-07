package envpanel

import (
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
)

// maskWidth is the fixed number of dots a hidden value renders as, so the
// masked column reveals neither the value nor its length.
const maskWidth = 8

func (m Model) View() tea.View {
	// The Env panel is the sole component on its page, so it is always the
	// focused tier - there is no second panel for Tab to move to. Same choice
	// composefilepanel makes.
	bg := chrome.PanelBg(true)

	bodyWidth := max(1, chrome.PanelBodyWidth(m.panelWidth))
	bodyAvail := max(1, chrome.PanelBodyHeight(m.panelHeight))

	var body string
	switch {
	case m.loading:
		body = chrome.EmptyCard(bodyWidth, bodyAvail, bg,
			"Loading .env",
			"Reading the environment file…",
			"", "")

	case m.loadErr != nil:
		body = chrome.EmptyCard(bodyWidth, bodyAvail, bg,
			"Could not read .env",
			m.loadErr.Error(),
			"E", "edit in $EDITOR")

	case len(m.entries) == 0:
		body = chrome.EmptyCard(bodyWidth, bodyAvail, bg,
			"No variables",
			"This .env file has no variables yet.",
			"n", "add a variable")

	default:
		body = m.renderTable(bodyWidth, bodyAvail, bg)
	}

	// The title carries the .env path on its right, the same way the Files
	// panel pins the compose file path: it answers "which file am I looking
	// at?" without spending a body row on it.
	titleRight := ""
	if m.envPath != "" {
		titleRight = lipgloss.NewStyle().
			Foreground(appstyles.Active.TextDim).
			Render(m.envPath)
	}

	screen := chrome.PanelFrame("Env", titleRight, true, m.panelWidth, m.panelHeight, body)
	return tea.NewView(screen)
}

// keyColWidth splits the row into a key column and a value column, capping
// the key at 32 columns so long keys don't starve the value.
func keyColWidth(contentWidth int) (int, int) {
	keyWidth := min(32, contentWidth/2)
	keyWidth = max(keyWidth, 8)
	valWidth := max(1, contentWidth-keyWidth-1) // -1 for the gap between columns
	return keyWidth, valWidth
}

// renderTable renders the KEY / VALUE table for the loaded entries, sized to
// fill the panel body so short files still seal the tier below the last row.
func (m Model) renderTable(bodyWidth, bodyAvail int, bg color.Color) string {
	parts := []string{
		m.renderHeader(bodyWidth),
		chrome.PanelRule(bodyWidth),
	}
	for i, entry := range m.entries {
		parts = append(parts, m.renderRow(i, entry, bodyWidth))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return chrome.PanelBodyWithFooter(bodyWidth, bodyAvail, bg, content, "")
}

// renderHeader is the dim, bold KEY / VALUE heading, laid out on the same
// column grid the rows use. The leading blank column matches the accent bar
// the selected row reserves, so the heading and the values line up.
func (m Model) renderHeader(bodyWidth int) string {
	contentWidth := max(1, bodyWidth-1)
	keyWidth, valWidth := keyColWidth(contentWidth)

	dim := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).Bold(true)

	return lipgloss.JoinHorizontal(lipgloss.Left,
		lipgloss.NewStyle().Width(1).Render(""),
		dim.Width(keyWidth + 1).Render("KEY"),
		dim.Width(valWidth).Render("VALUE"),
	)
}

// renderRow renders one entry. Variable rows are a two-column key/value row;
// comments, blank lines, and parse errors span the full width. The selected
// row is lifted to the surface tier with an accent bar down its left edge -
// the same selection language the service and group lists use.
func (m Model) renderRow(idx int, entry cmds.EnvEntry, bodyWidth int) string {
	isSelected := idx == m.selectedIdx
	rowBg := chrome.ListRowBg(isSelected, true)

	contentWidth := max(1, bodyWidth-1)

	var content string
	switch entry.Source {
	case "comment":
		content = lipgloss.NewStyle().
			Foreground(appstyles.Active.TextDim).
			Background(rowBg).
			Width(contentWidth).
			Render(chrome.Truncate(entry.Raw, contentWidth))

	case "parse_error":
		content = lipgloss.NewStyle().
			Foreground(appstyles.Active.Danger).
			Background(rowBg).
			Width(contentWidth).
			Render(chrome.Truncate("[parse error] "+entry.Raw, contentWidth))

	case "var":
		keyWidth, valWidth := keyColWidth(contentWidth)

		keyColor := appstyles.Active.TextPrimary
		if !isSelected {
			keyColor = appstyles.Active.TextMuted
		}

		value := strings.Repeat("•", maskWidth)
		if m.IsRevealed(idx) {
			value = entry.Value
		}

		keyCell := lipgloss.NewStyle().
			Foreground(keyColor).
			Background(rowBg).
			Width(keyWidth + 1). // +1 for the gap to the value column
			Render(chrome.Truncate(entry.Key, keyWidth))

		valCell := lipgloss.NewStyle().
			Foreground(appstyles.Active.TextPrimary).
			Background(rowBg).
			Width(valWidth).
			Render(chrome.Truncate(value, valWidth))

		content = lipgloss.JoinHorizontal(lipgloss.Left, keyCell, valCell)

	default: // "blank" and anything unrecognised: a full-width empty row
		content = lipgloss.NewStyle().
			Background(rowBg).
			Width(contentWidth).
			Render("")
	}

	// The accent bar marks the cursor row; other rows reserve the same column
	// in the row background so every row lines up on the same left edge.
	barColor := rowBg
	if isSelected {
		barColor = appstyles.Active.Accent
	}
	bar := chrome.BarColumn(barColor, rowBg, content)

	row := lipgloss.JoinHorizontal(lipgloss.Left, bar, content)

	// Seal the row to its own tier: the active row's lighter surface must not
	// be flattened to the panel tier by a shorter styled run below it.
	return appstyles.FillBackground(rowBg, row)
}
