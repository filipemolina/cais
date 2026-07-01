package components

import (
	"image/color"
	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/apptypes"
	"stack-stitcher/src/cmds"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var menuItemStyle = lipgloss.NewStyle().
	Foreground(appstyles.PrimaryFontColor).
	Padding(0, 2)

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
				}

			case "right", "l":
				if m.focusedItemIndex < len(m.items)-1 {
					m.focusedItemIndex++
				}

			case "space":
				m.selectedItemIndex = m.focusedItemIndex
			}
		}
	}

	return m, nil
}

func (m MainMenuModel) View() tea.View {
	var renderedItems []string
	var borderColor color.Color

	if m.isFocused {
		borderColor = appstyles.PrimaryColor
	} else {
		borderColor = lipgloss.Darken(appstyles.SecondaryFontColor, 0.5)
	}

	for index, item := range m.items {
		prefix := "  "
		sufix := prefix

		itemStyle := menuItemStyle.
			BorderStyle(appstyles.InactiveTabBorder).
			BorderForeground(borderColor)

		if m.isFocused && index == m.focusedItemIndex {
			prefix = "> "
		}

		if index == m.selectedItemIndex {
			itemStyle = itemStyle.
				Bold(true).
				BorderStyle(appstyles.ActiveTabBorder)
		}

		if !m.isFocused && index != m.selectedItemIndex {
			itemStyle = itemStyle.Foreground(lipgloss.Darken(appstyles.PrimaryFontColor, 0.5))
		}

		renderedItems = append(renderedItems, itemStyle.Render(prefix+item+sufix))
	}

	items := lipgloss.JoinHorizontal(lipgloss.Top, renderedItems...)
	tabsWidth := lipgloss.Width(items)

	gapWidth := (m.terminalWidth - tabsWidth)
	gapStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(borderColor).
		Width(gapWidth).
		PaddingBottom(1)

	gap := gapStyle.Render("")

	tabBar := lipgloss.JoinHorizontal(lipgloss.Left, items, gap)

	return tea.NewView(tabBar)
}

func MainMenu() tea.Model {
	items := []string{}

	for _, page := range apptypes.PageTitles {
		items = append(items, page)
	}

	m := MainMenuModel{items: items}

	return m
}
