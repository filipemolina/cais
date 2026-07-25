package components

import (
	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/apptypes"
	"stack-stitcher/src/cmds"
	"stack-stitcher/src/constants"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type MainMenuModel struct {
	items             []string
	focusedItemIndex  int
	selectedItemIndex int
	terminalWidth     int
	terminalHeight    int
	isFocused         bool
	componentId       int
}

func (m MainMenuModel) Init() tea.Cmd {
	return nil
}

func (m MainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var finalCmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height

	case cmds.SetFocusMsg:
		if int(msg) == m.componentId {
			m.isFocused = true
		} else {
			m.isFocused = false
		}

	case tea.KeyPressMsg:
		if m.isFocused {
			switch msg.String() {
			case "left", "h":
				if m.focusedItemIndex > 0 {
					m.focusedItemIndex--
					m.selectedItemIndex = m.focusedItemIndex
					pageTitle := apptypes.PageTitles[m.focusedItemIndex]
					setPageCmd := cmds.SetActivePage(pageTitle)
					finalCmds = append(finalCmds, setPageCmd)
				}

			case "right", "l":
				if m.focusedItemIndex < len(m.items)-1 {
					m.focusedItemIndex++
					m.selectedItemIndex = m.focusedItemIndex
					pageTitle := apptypes.PageTitles[m.focusedItemIndex]
					setPageCmd := cmds.SetActivePage(pageTitle)
					finalCmds = append(finalCmds, setPageCmd)
				}

			case "space":
				m.selectedItemIndex = m.focusedItemIndex
				pageTitle := apptypes.PageTitles[m.focusedItemIndex]
				setPageCmd := cmds.SetActivePage(pageTitle)
				finalCmds = append(finalCmds, setPageCmd)
			}
		}
	}

	return m, tea.Batch(finalCmds...)
}

func (m MainMenuModel) View() tea.View {
	// The whole nav sits on tier 2 background. No bottom border — the
	// tier 2 vs tier 3/4 background contrast handles the section break.
	navStyle := lipgloss.NewStyle().
		Background(appstyles.BackgroundContent).
		Width(m.terminalWidth)

	// Active tab: bold white text on tier 2, with a thick Accent-colored
	// underline. A `▌` accent bar in front of the text.
	activeTabStyle := lipgloss.NewStyle().
		Foreground(appstyles.TextPrimary).
		Bold(true).
		Background(appstyles.BackgroundContent).
		Padding(0, 2, 0, 1) // less left padding to compensate for the external ▌

	// Focused-by-Tab (but not the selected page): white text, no underline.
	focusedTabStyle := lipgloss.NewStyle().
		Foreground(appstyles.TextPrimary).
		Background(appstyles.BackgroundContent).
		Padding(0, 2)

	// Inactive: dim text on tier 2.
	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(appstyles.TextDim).
		Background(appstyles.BackgroundContent).
		Padding(0, 2)

	accentBar := lipgloss.NewStyle().Foreground(appstyles.Accent).Render("▌")

	// Wordmark badge at the far left, using the accent color on the
	// same tier-2 bar background.
	wordmarkStyle := lipgloss.NewStyle().
		Foreground(appstyles.Accent).
		Background(appstyles.BackgroundContent).
		Bold(true).
		Padding(0, 1)

	var cells []string
	for index, item := range m.items {
		label := apptypes.PageLabels[item]
		if label == "" {
			label = item
		}

		isSelected := index == m.selectedItemIndex
		isTabFocused := m.isFocused && index == m.focusedItemIndex

		var cell string
		switch {
		case isSelected:
			cell = lipgloss.JoinHorizontal(lipgloss.Left, accentBar, activeTabStyle.Render(label))
		case isTabFocused:
			cell = focusedTabStyle.Render(label)
		default:
			cell = inactiveTabStyle.Render(label)
		}
		cells = append(cells, cell)
	}

	badge := wordmarkStyle.Render(constants.WORDMARK)
	tabs := lipgloss.JoinHorizontal(lipgloss.Left, cells...)
	menuRow := lipgloss.JoinHorizontal(lipgloss.Left, badge, tabs)
	nav := navStyle.Render(menuRow)

	return tea.NewView(nav)
}

func MainMenu() tea.Model {
	items := []string{}

	for _, page := range apptypes.PageTitles {
		items = append(items, page)
	}

	m := MainMenuModel{items: items}

	return m
}
