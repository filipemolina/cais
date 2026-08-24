package envmodal

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/keys"
)

// maskWidth is the fixed number of dots a hidden value renders as, so the
// masked column reveals neither the value nor its length.
const maskWidth = 8

// modalWidth is the column width the table renders at inside the modal. The
// modal surface adds its own padding around this.
const modalWidth = 60

func (m Model) View() tea.View {
	var body string

	switch {
	case m.loading:
		body = chrome.ModalHints(
			chrome.HintFor(keys.Overlay.Cancel),
		)
		content := lipgloss.JoinVertical(lipgloss.Left,
			chrome.ModalTitle("Env"),
			"Loading .env…",
			"",
			body,
		)
		return tea.NewView(chrome.ModalSurface(appstyles.Active.ModalBg, content))

	case m.loadErr != nil:
		content := lipgloss.JoinVertical(lipgloss.Left,
			chrome.ModalTitle("Env"),
			lipgloss.NewStyle().Foreground(appstyles.Active.Danger).Render(m.loadErr.Error()),
			"",
			chrome.ModalHints(chrome.HintFor(keys.Overlay.Cancel)),
		)
		return tea.NewView(chrome.ModalSurface(appstyles.Active.ModalBg, content))

	case len(m.entries) == 0:
		content := lipgloss.JoinVertical(lipgloss.Left,
			chrome.ModalTitle("Env"),
			"This .env file has no variables yet.",
			"",
			chrome.ModalHints(
				chrome.HintFor(keys.List.New),
				chrome.HintFor(keys.Overlay.Cancel),
			),
		)
		return tea.NewView(chrome.ModalSurface(appstyles.Active.ModalBg, content))

	default:
		body = m.renderTable(modalWidth)
	}

	hints := chrome.ModalHints(
		chrome.HintFor(keys.List.New),
		chrome.HintFor(keys.List.Edit),
		chrome.HintFor(keys.List.Delete),
		chrome.HintFor(key.NewBinding(key.WithHelp("space", "reveal"))),
		chrome.HintFor(key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy"))),
		chrome.HintFor(key.NewBinding(key.WithKeys("E"), key.WithHelp("E", "editor"))),
		chrome.HintFor(key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "raw"))),
		chrome.HintFor(keys.Overlay.Cancel),
	)

	var header string
	if m.parseErrorCount > 0 {
		header = lipgloss.NewStyle().
			Foreground(appstyles.Active.Danger).
			Render(fmt.Sprintf("%d parse error%s", m.parseErrorCount, plural(m.parseErrorCount)))
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		chrome.ModalTitle("Env"),
		header,
		body,
		"",
		hints,
	)

	return tea.NewView(chrome.ModalSurface(appstyles.Active.ModalBg, content))
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// keyColWidth splits the row into a key column and a value column, capping
// the key at 28 columns so long keys don't starve the value.
func keyColWidth(contentWidth int) (int, int) {
	keyWidth := min(28, contentWidth/2)
	keyWidth = max(keyWidth, 8)
	valWidth := max(1, contentWidth-keyWidth-1) // -1 for the gap between columns
	return keyWidth, valWidth
}

// renderTable renders the KEY / VALUE table for the loaded entries.
func (m Model) renderTable(contentWidth int) string {
	parts := []string{
		m.renderHeader(contentWidth),
		chrome.PanelRule(contentWidth),
	}
	for i, entry := range m.entries {
		parts = append(parts, m.renderRow(i, entry, contentWidth))
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) renderHeader(contentWidth int) string {
	dim := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).Bold(true)

	return lipgloss.JoinHorizontal(lipgloss.Left,
		lipgloss.NewStyle().Width(1).Render(""),
		dim.Width(28+1).Render("KEY"),
		dim.Width(max(1, contentWidth-29)).Render("VALUE"),
	)
}

// renderRow renders one entry. Variable rows are a two-column key/value row;
// comments, blank lines, and parse errors span the full width. The selected
// row is lifted to the surface tier with an accent bar down its left edge -
// the same selection language the env/file rows use.
func (m Model) renderRow(idx int, entry cmds.EnvEntry, contentWidth int) string {
	isSelected := idx == m.selectedIdx
	rowBg := chrome.ListRowBg(isSelected)

	var rowContent string
	switch entry.Source {
	case "comment":
		rowContent = lipgloss.NewStyle().
			Foreground(appstyles.Active.TextDim).
			Background(rowBg).
			Width(contentWidth).
			Render(chrome.Truncate(entry.Raw, contentWidth))

	case "parse_error":
		rowContent = lipgloss.NewStyle().
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
		if m.revealedIdx == idx {
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

		rowContent = lipgloss.JoinHorizontal(lipgloss.Left, keyCell, valCell)

	default: // "blank" and anything unrecognised: a full-width empty row
		rowContent = lipgloss.NewStyle().
			Background(rowBg).
			Width(contentWidth).
			Render("")
	}

	// The accent bar marks the cursor row; other rows reserve the same
	// column in the row background so every row lines up on the same edge.
	barColor := rowBg
	if isSelected {
		barColor = appstyles.Active.Accent
	}
	bar := chrome.BarColumn(barColor, rowBg, rowContent)

	row := lipgloss.JoinHorizontal(lipgloss.Left, bar, rowContent)

	return appstyles.FillBackground(rowBg, row)
}
