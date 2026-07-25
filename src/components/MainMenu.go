package components

import (
	"image/color"
	"slices"
	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/apptypes"
	"stack-stitcher/src/cmds"
	"stack-stitcher/src/constants"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// MainMenuModel is the top nav bar. It is not focusable and handles no keys:
// pages are switched with the global alt+<letter> chords that it advertises by
// underlining each label's first letter (see apptypes.PageShortcut). All it
// tracks is which page is active, so it can highlight that tab.
type MainMenuModel struct {
	items             []string
	selectedItemIndex int
	terminalWidth     int
}

func (m MainMenuModel) Init() tea.Cmd {
	return nil
}

func (m MainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width

	case cmds.SetActivePageMsg:
		if idx := slices.Index(m.items, string(msg)); idx >= 0 {
			m.selectedItemIndex = idx
		}
	}

	return m, nil
}

// tabLabel renders a page label with its first letter underlined. That letter
// is the page's alt+<letter> chord (see apptypes.PageShortcut), so underlining
// it advertises the shortcut in place rather than spending a separate hint on
// each page.
func tabLabel(label string, fg color.Color, bold bool) string {
	runes := []rune(label)
	if len(runes) == 0 {
		return label
	}

	base := lipgloss.NewStyle().
		Foreground(fg).
		Background(appstyles.BackgroundContent).
		Bold(bold)

	return base.Underline(true).Render(string(runes[0])) + base.Render(string(runes[1:]))
}

func (m MainMenuModel) View() tea.View {
	// The whole nav sits on tier 2 background. No bottom border — the
	// tier 2 vs tier 3/4 background contrast handles the section break.
	navStyle := lipgloss.NewStyle().
		Background(appstyles.BackgroundContent).
		Width(m.terminalWidth)

	// Cell styles carry only the spacing. All text styling - color, bold, and
	// the shortcut underline - happens in tabLabel, so the underline is not
	// competing with a foreground set on the enclosing style.
	cellStyle := lipgloss.NewStyle().
		Background(appstyles.BackgroundContent).
		Padding(0, 2)

	// The active cell has less left padding to compensate for the external ▌.
	activeCellStyle := cellStyle.Padding(0, 2, 0, 1)

	accentBar := lipgloss.NewStyle().
		Foreground(appstyles.Accent).
		Background(appstyles.BackgroundContent).
		Render("▌")

	// Wordmark badge, now at the far right, in the accent color on the same
	// tier-2 bar background.
	wordmarkStyle := lipgloss.NewStyle().
		Foreground(appstyles.Accent).
		Background(appstyles.BackgroundContent).
		Bold(true).
		Padding(0, 2)

	var cells []string
	for index, item := range m.items {
		isSelected := index == m.selectedItemIndex

		if isSelected {
			cell := activeCellStyle.Render(tabLabel(apptypes.PageLabel(item), appstyles.TextPrimary, true))
			cells = append(cells, lipgloss.JoinHorizontal(lipgloss.Left, accentBar, cell))
			continue
		}

		cells = append(cells, cellStyle.Render(tabLabel(apptypes.PageLabel(item), appstyles.TextDim, false)))
	}

	tabs := lipgloss.JoinHorizontal(lipgloss.Left, cells...)
	badge := wordmarkStyle.Render(constants.WORDMARK)

	// Push the wordmark to the far right. The gap carries the bar background so
	// the whole row stays tier 2; navStyle's own Width would only pad after the
	// badge, leaving it stuck to the tabs.
	gapWidth := max(0, m.terminalWidth-lipgloss.Width(tabs)-lipgloss.Width(badge))
	gap := lipgloss.NewStyle().
		Background(appstyles.BackgroundContent).
		Width(gapWidth).
		Render("")

	menuRow := lipgloss.JoinHorizontal(lipgloss.Left, tabs, gap, badge)

	return tea.NewView(navStyle.Render(menuRow))
}

func MainMenu() tea.Model {
	items := []string{}

	for _, page := range apptypes.PageTitles {
		items = append(items, page)
	}

	m := MainMenuModel{items: items}

	return m
}
