package model

import (
	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/constants"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var errorBannerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FAFAFA")).
	Background(lipgloss.Color("#B33A3A")).
	Padding(0, 1)

func (m AppModel) View() tea.View {
	var v tea.View
	mainMenu := m.components.MainMenu.View().Content
	pageComponents, ok := m.pages[m.activePage]

	if ok {
		var contents []string

		for idx, _ := range pageComponents {
			contents = append(contents, pageComponents[idx].View().Content)
		}

		// Add a thin tier 2 gutter between panels so they don't touch. Its
		// width is the same constant AppModel subtracts from the row before
		// sizing the panels, so the three pieces add up to the terminal
		// width exactly. Before the first WindowSizeMsg the broadcast height
		// is 0; fall back to the tallest rendered panel so the gutter still
		// spans the body.
		bodyHeight := m.config.bodyLayout.Height
		if bodyHeight == 0 {
			for _, c := range contents {
				if h := lipgloss.Height(c); h > bodyHeight {
					bodyHeight = h
				}
			}
		}
		gutter := lipgloss.NewStyle().
			Background(appstyles.BackgroundContent).
			Width(constants.BODY_GUTTER_WIDTH).
			Height(bodyHeight).
			Render("")

		var bodyParts []string
		for i, c := range contents {
			bodyParts = append(bodyParts, c)
			if i < len(contents)-1 {
				bodyParts = append(bodyParts, gutter)
			}
		}
		body := lipgloss.JoinHorizontal(lipgloss.Top, bodyParts...)

		keybindingBar := m.components.KeybindingBar.View().Content

		sections := []string{mainMenu, body, keybindingBar}
		if m.lastError != "" {
			sections = []string{errorBannerStyle.Render("Error: " + m.lastError), mainMenu, body, keybindingBar}
		}

		layout := lipgloss.JoinVertical(lipgloss.Left, sections...)

		// Wrap the full layout in a style that fills the terminal width
		// with the tier-2 background.  JoinVertical pads narrower sections
		// (nav bar, footer) with plain spaces when the body panels are
		// wider; without an explicit background those padding characters
		// show the terminal default color, creating thin horizontal
		// divider lines at the section boundaries.
		//
		// MaxWidth/MaxHeight are the backstop: Width() pads but never
		// truncates, so any section that renders wider than the terminal
		// would otherwise be wrapped by the terminal itself, scattering the
		// overflow across the following lines.
		rendered := lipgloss.NewStyle().
			Background(appstyles.BackgroundContent).
			Width(m.config.terminalWidht).
			Height(m.config.terminalHeight).
			MaxWidth(m.config.terminalWidht).
			MaxHeight(m.config.terminalHeight).
			Render(layout)

		if m.activeModal != nil {
			rendered = m.renderWithModal(rendered)
		}

		v = tea.NewView(rendered)
		v.AltScreen = true
	}

	return v
}

// renderWithModal composites the active modal as a centered layer on top
// of the rest of the screen.
func (m AppModel) renderWithModal(base string) string {
	modalContent := m.activeModal.View().Content

	x := max(0, (m.config.terminalWidht-lipgloss.Width(modalContent))/2)
	y := max(0, (m.config.terminalHeight-lipgloss.Height(modalContent))/2)

	baseLayer := lipgloss.NewLayer(base)
	modalLayer := lipgloss.NewLayer(modalContent).X(x).Y(y).Z(1)

	return lipgloss.NewCompositor(baseLayer, modalLayer).Render()
}
