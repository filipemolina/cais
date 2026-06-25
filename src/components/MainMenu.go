package components

import (
	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/apptypes"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var backGroundColor = lipgloss.Darken(lipgloss.Color(appstyles.PrimaryColor), 0.5)
var focusedBackgroundColor = lipgloss.Color(appstyles.PrimaryColor)

var menuWrapperStyle = lipgloss.NewStyle().
	Background(backGroundColor)

var menuItemStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#FAFAFA")).
	Background(backGroundColor).
	Padding(0, 2)

var focusedMenuItemStyle = menuItemStyle.
	Background(focusedBackgroundColor)

type MainMenuModel struct {
	items            []string
	focusedItemIndex int
	terminalWidth    int
	terminalHeight   int
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
		}
	}

	return m, nil
}

func (m MainMenuModel) View() tea.View {
	var renderedItems []string

	for index, item := range m.items {
		if index == m.focusedItemIndex {
			renderedItems = append(renderedItems, focusedMenuItemStyle.Render(item))
		} else {
			renderedItems = append(renderedItems, menuItemStyle.Render(item))
		}
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
