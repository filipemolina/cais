package mainmenu

import (
	"fmt"
	"image/color"
	"slices"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/constants"
)

// Model is the top nav bar. It is not focusable and handles no keys:
// pages are switched with the global digit keys that it advertises by
// rendering each tab's digit before its label. All it tracks is which page is
// active, so it can highlight that tab.
type Model struct {
	items             []string
	selectedItemIndex int
	terminalWidth     int
}

// versionGutter is the breathing room the version needs between the last tab
// and the wordmark. Below it the two would read as one word, so the version
// is dropped instead.
const versionGutter = 4

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

// tabLabel renders a tab as "<digit> <label>". The digit is the page's global
// shortcut - 1 is the first page - shown in the accent color so the nav
// advertises the key in place rather than spending a footer hint on each page.
// It replaced the first-letter underline when digits became the primary
// scheme; the alt+<letter> chord remains as an alias (apptypes.PageShortcut).
func tabLabel(page string, index int, fg color.Color, bold bool) string {
	digit := lipgloss.NewStyle().
		Foreground(appstyles.Active.Accent).
		Background(appstyles.Active.BackgroundContent).
		Bold(true).
		Render(fmt.Sprintf("%d", index+1))

	label := lipgloss.NewStyle().
		Foreground(fg).
		Background(appstyles.Active.BackgroundContent).
		Bold(bold).
		Render(" " + apptypes.PageLabel(page))

	return digit + label
}

func (m Model) View() tea.View {
	// The whole nav sits on tier 2 background. No bottom border — the
	// tier 2 vs tier 3/4 background contrast handles the section break.
	navStyle := lipgloss.NewStyle().
		Background(appstyles.Active.BackgroundContent).
		Width(m.terminalWidth)

	// Cell styles carry only the spacing. All text styling - color, bold, and
	// the accent digit - happens in tabLabel, so the digit is not competing
	// with a foreground set on the enclosing style.
	cellStyle := lipgloss.NewStyle().
		Background(appstyles.Active.BackgroundContent).
		Padding(0, 2)

	// The active cell has less left padding to compensate for the external ▌.
	activeCellStyle := cellStyle.Padding(0, 2, 0, 1)

	accentBar := lipgloss.NewStyle().
		Foreground(appstyles.Active.Accent).
		Background(appstyles.Active.BackgroundContent).
		Render("▌")

	// Wordmark badge, now at the far right, in the accent color on the same
	// tier-2 bar background.
	wordmarkStyle := lipgloss.NewStyle().
		Foreground(appstyles.Active.Accent).
		Background(appstyles.Active.BackgroundContent).
		Bold(true).
		Padding(0, 2)

	var cells []string
	for index, item := range m.items {
		isSelected := index == m.selectedItemIndex

		if isSelected {
			cell := activeCellStyle.Render(tabLabel(item, index, appstyles.Active.TextPrimary, true))
			cells = append(cells, lipgloss.JoinHorizontal(lipgloss.Left, accentBar, cell))
			continue
		}

		cells = append(cells, cellStyle.Render(tabLabel(item, index, appstyles.Active.TextDim, false)))
	}

	tabs := lipgloss.JoinHorizontal(lipgloss.Left, cells...)
	badge := wordmarkStyle.Render(constants.WORDMARK)

	// The version rides just left of the wordmark, dimmed - it answers "which
	// build is this" for a bug report, and should never compete with the tabs
	// for attention. It is dropped rather than wrapped when the row is tight,
	// the same bargain the footer makes with the compose file name.
	versionStyle := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextDim).
		Background(appstyles.Active.BackgroundContent)

	version := versionStyle.Render(constants.Version())
	if lipgloss.Width(tabs)+lipgloss.Width(badge)+lipgloss.Width(version)+versionGutter > m.terminalWidth {
		version = ""
	}

	// Push the wordmark to the far right. The gap carries the bar background so
	// the whole row stays tier 2; navStyle's own Width would only pad after the
	// badge, leaving it stuck to the tabs.
	gapWidth := max(0, m.terminalWidth-lipgloss.Width(tabs)-lipgloss.Width(badge)-lipgloss.Width(version))
	gap := lipgloss.NewStyle().
		Background(appstyles.Active.BackgroundContent).
		Width(gapWidth).
		Render("")

	menuRow := lipgloss.JoinHorizontal(lipgloss.Left, tabs, gap, version, badge)

	return tea.NewView(navStyle.Render(menuRow))
}

// New builds the top nav bar.
func New() tea.Model {
	items := []string{}

	for _, page := range apptypes.PageTitles {
		items = append(items, page)
	}

	m := Model{items: items}

	return m
}
