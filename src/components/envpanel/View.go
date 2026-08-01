package envpanel

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/cmds"
)

const maskWidth = 8 // Fixed-width mask: •••••••• (8 chars)

func (m Model) View() tea.View {
	if m.loadErr != nil {
		body := fmt.Sprintf("Error: %v", m.loadErr)
		return tea.NewView(body)
	}

	if m.loading {
		body := "Loading .env..."
		return tea.NewView(body)
	}

	var body string
	if len(m.entries) == 0 {
		body = lipgloss.NewStyle().
			Foreground(appstyles.Active.TextDim).
			Width(m.panelWidth).
			AlignHorizontal(lipgloss.Center).
			Render("No .env file or file is empty\nPress n to add the first variable")
	} else {
		// Build the table
		header := m.renderTableHeader()
		divider := chrome.PanelRule(m.panelWidth)

		var rows []string
		for i, entry := range m.entries {
			rows = append(rows, m.renderRow(i, entry))
		}

		parts := []string{header, divider}
		parts = append(parts, rows...)

		body = lipgloss.NewStyle().
			Width(m.panelWidth).
			Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
	}

	return tea.NewView(body)
}

func (m Model) renderTableHeader() string {
	style := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextDim).
		PaddingLeft(1)

	return style.Render(fmt.Sprintf("%-30s VALUE", "KEY"))
}

func (m Model) renderRow(idx int, entry cmds.EnvEntry) string {
	isSelected := idx == m.selectedIdx
	isRevealed := m.IsRevealed(idx)

	switch entry.Source {
	case "blank":
		// Blank line - render as empty row
		return ""

	case "comment":
		// Comment line
		style := lipgloss.NewStyle().
			Foreground(appstyles.Active.TextDim)
		return style.Render(entry.Raw)

	case "parse_error":
		// Parse error
		style := lipgloss.NewStyle().
			Foreground(appstyles.Active.Danger).
			PaddingLeft(1)
		return style.Render(fmt.Sprintf("[Parse error] %s", entry.Raw))

	case "var":
		// Environment variable
		keyPart := entry.Key
		if len(keyPart) > 30 {
			keyPart = keyPart[:27] + "..."
		}
		keyPart = fmt.Sprintf("%-30s", keyPart)

		// Render value: masked or revealed
		valuePart := ""
		if isRevealed {
			valuePart = entry.Value
		} else {
			// Use fixed-width mask
			valuePart = strings.Repeat("•", maskWidth)
		}

		// Build the row
		row := keyPart + valuePart

		// Apply selection styling
		if isSelected {
			return lipgloss.NewStyle().
				Foreground(appstyles.Active.PanelBg).
				Background(appstyles.Active.Accent).
				PaddingLeft(1).
				Render(row)
		}

		return lipgloss.NewStyle().
			PaddingLeft(1).
			Render(row)

	default:
		return ""
	}
}
