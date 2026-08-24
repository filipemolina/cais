package composefilepanel

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/components/chrome"
)

// errorContent renders a read failure as plain text the viewport can
// display when the file could not be loaded.
func errorContent(message string) string {
	return lipgloss.NewStyle().
		Foreground(appstyles.Active.Danger).
		Render("Error: " + message)
}

func (m Model) View() tea.View {
	// Always the focused tier: see the note on the model.
	bg := chrome.PanelBg()

	bodyWidth := max(1, chrome.PanelBodyWidth(m.panelWidth))
	bodyAvail := max(1, chrome.PanelBodyHeight(m.panelHeight))

	var body string
	switch {
	case m.filePath == "":
		body = chrome.EmptyCard(bodyWidth, bodyAvail, bg,
			"No compose file",
			"No compose file is loaded. Run Cais from a directory with a compose file, or use --file/--dir.",
			"", "")
	case m.readErr != nil:
		body = chrome.EmptyCard(bodyWidth, bodyAvail, bg,
			"Could not read file",
			m.readErr.Error(),
			"E", "edit in $EDITOR")
	case m.loaded && strings.TrimSpace(m.content) == "":
		body = chrome.EmptyCard(bodyWidth, bodyAvail, bg,
			"Empty file",
			"The compose file is empty.",
			"E", "edit in $EDITOR")
	default:
		vp := m.viewport.View()
		// Seal the viewport against the panel background. JoinVertical below
		// would pad the shorter rows out to the panel width with unstyled
		// spaces, so the viewport output must already carry the background.
		vp = appstyles.FillBackground(bg, vp)
		body = lipgloss.NewStyle().
			Width(bodyWidth).
			Height(bodyAvail).
			MaxHeight(bodyAvail).
			Background(bg).
			Render(vp)
	}

	// The title shows the file path when one is loaded. The path is the
	// answer to "which file am I looking at?" and earns its place on the
	// title row the same way a status pill does on the details panels.
	title := "Files"
	titleRight := ""
	if m.filePath != "" {
		titleRight = lipgloss.NewStyle().
			Foreground(appstyles.Active.TextDim).
			Render(m.filePath)
	}

	screen := chrome.PanelFrame(title, titleRight, m.panelWidth, m.panelHeight, body)
	return tea.NewView(screen)
}
