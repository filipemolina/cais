package model

import (
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/constants"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var errorBannerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FAFAFA")).
	Background(lipgloss.Color("#B33A3A")).
	Padding(0, 1)

func (m AppModel) View() tea.View {
	mainMenu := m.components.MainMenu.View().Content
	keybindingBar := m.components.KeybindingBar.View().Content

	sections := []string{mainMenu, m.renderBody(), keybindingBar}
	if m.lastError != "" {
		sections = append([]string{errorBannerStyle.Render("Error: " + m.lastError)}, sections...)
	}

	layout := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Seal the frame against tier 2. JoinVertical pads the narrower
	// sections (nav bar, footer, error banner) out to the body width with
	// unstyled spaces, and the outer style below cannot fix that - it only
	// paints the padding it adds itself.
	//
	// This is the outermost tier, so it must run last: every inner tier has
	// already sealed its own region, which leaves no unpainted cell inside
	// a panel for this pass to reach. See appstyles.FillBackground.
	layout = appstyles.FillBackground(appstyles.BackgroundContent, layout)

	// Wrap the full layout in a style that fills the terminal width
	// with the tier-2 background.
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

	// AltScreen is set unconditionally. Returning the zero tea.View - which is
	// what this used to do for a page missing from the pages map - leaves
	// AltScreen false, dropping the terminal back to its normal buffer. The app
	// keeps running but goes invisible, which reads as a crash.
	v := tea.NewView(rendered)
	v.AltScreen = true

	return v
}

// renderBody renders the active page's panels side by side, separated by the
// tier-2 gutter. A page with no registered components yields an empty body box
// of the right size rather than nothing, so the frame keeps its shape.
func (m AppModel) renderBody() string {
	pageComponents := m.pages[m.activePage]

	var contents []string
	for idx := range pageComponents {
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

	if len(contents) == 0 {
		return lipgloss.NewStyle().
			Background(appstyles.BackgroundContent).
			Width(m.config.terminalWidht).
			Height(bodyHeight).
			Render("")
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

	return lipgloss.JoinHorizontal(lipgloss.Top, bodyParts...)
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
