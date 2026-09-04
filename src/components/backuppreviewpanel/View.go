package backuppreviewpanel

import (
	"fmt"
	"image/color"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/components/chrome"
)

func (m Model) View() tea.View {
	bg := chrome.PanelBg()

	bodyWidth := max(1, chrome.PanelBodyWidth(m.panelWidth))
	bodyAvail := max(1, chrome.PanelBodyHeight(m.panelHeight))

	var body string
	switch {
	case !m.hasSelection():
		body = chrome.EmptyCard(bodyWidth, bodyAvail, bg,
			"Nothing selected",
			"Pick a version on the left to see what it holds.",
			"", "")

	case m.loadErr != nil:
		body = chrome.EmptyCard(bodyWidth, bodyAvail, bg,
			"Could not read this copy",
			m.loadErr.Error(),
			"", "")

	default:
		body = m.renderContent(bodyWidth, bodyAvail, bg)
	}

	titleRight := ""
	if m.hasSelection() {
		titleRight = lipgloss.NewStyle().
			Foreground(appstyles.Active.TextDim).
			Render(fmt.Sprintf("%s · %s", m.entry.File, m.entry.SHA8))
	}

	screen := chrome.PanelFrame("Preview", titleRight, m.panelWidth, m.panelHeight, body)
	return tea.NewView(screen)
}

// renderContent puts the viewport in the panel body under a header naming
// the copy's write time, so the panel says which version is on screen
// without the reader having to look back at the list.
func (m Model) renderContent(width, avail int, bg color.Color) string {
	header := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextDim).
		Background(bg).
		Width(width).
		Render(m.entry.Timestamp.UTC().Format("2006-01-02 15:04") + " UTC")

	vp := appstyles.FillBackground(bg, m.vp.View())

	// Clip the viewport to the height left under the header row.
	vp = lipgloss.NewStyle().MaxHeight(max(0, avail-1)).Render(vp)

	content := lipgloss.JoinVertical(lipgloss.Left, header, vp)
	return chrome.PanelBodyWithFooter(width, avail, bg, content, "")
}
