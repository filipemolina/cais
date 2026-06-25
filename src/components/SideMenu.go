package components

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var listStyle = lipgloss.NewStyle().Margin(1, 2)

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type SideMenuModel struct {
	list list.Model
}

func (m SideMenuModel) Init() tea.Cmd {
	return nil
}

func (m SideMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := listStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m SideMenuModel) View() tea.View {
	v := tea.NewView(listStyle.Render(m.list.View()))
	return v
}

func SideMenu() tea.Model {
	items := []list.Item{
		item{title: "Home", desc: "View running services"},
		item{title: "Groups", desc: "Manage service groups"},
		item{title: "Options", desc: "Settings  and  options"},
	}

	list := list.New(items, list.NewDefaultDelegate(), 0, 0)
	list.Title = "Main Menu"

	m := SideMenuModel{list: list}

	return m
}
