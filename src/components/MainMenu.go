package components

import (
	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/apptypes"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var menuWrapperStyle = lipgloss.NewStyle().
	Background(appstyles.BackgroundColor)

var menuItemStyle = lipgloss.NewStyle().
	Foreground(appstyles.PrimaryFontColor).
	Background(appstyles.BackgroundColor).
	Padding(0, 2)

type MainMenuModel struct {
	items             []string
	focusedItemIndex  int
	selectedItemIndex int
	terminalWidth     int
	terminalHeight    int
}

func (m MainMenuModel) Init() tea.Cmd {
	return nil
}

func (m MainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height

	case tea.KeyPressMsg:

		switch msg.String() {
		case "left":
			if m.focusedItemIndex > 0 {
				m.focusedItemIndex--
			}

		case "right":
			if m.focusedItemIndex < len(m.items)-1 {
				m.focusedItemIndex++
			}

		case "space":
			m.selectedItemIndex = m.focusedItemIndex
		}
	}

	return m, nil
}

func (m MainMenuModel) View() tea.View {
	var renderedItems []string

	for index, item := range m.items {
		var itemStyle = menuItemStyle

		if index == m.selectedItemIndex {
			itemStyle = itemStyle.Bold(true).Underline(true)
		}

		if index == m.focusedItemIndex {
			itemStyle = itemStyle.Background(appstyles.FocusedBackgroundColor)
		}

		renderedItems = append(renderedItems, itemStyle.Render(item))
	}

	items := lipgloss.JoinHorizontal(lipgloss.Top, renderedItems...)

	menu := menuWrapperStyle.
		Width(m.terminalWidth).
		Render(items)

	return tea.NewView(menu)
}

func MainMenu() tea.Model {
	items := []string{}

	for _, page := range apptypes.PageTitles {
		items = append(items, page)
	}

	m := MainMenuModel{items: items}

	return m
}
