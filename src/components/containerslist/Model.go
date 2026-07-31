package containerslist

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/apptypes"
)

type Model struct {
	list        list.Model
	isFocused   bool
	componentId int
}

func (m Model) Init() tea.Cmd {
	return nil
}

func New(containers []apptypes.ContainerListItem, width int, height int) tea.Model {
	var items []list.Item

	for _, container := range containers {
		items = append(items, container)
	}

	servicesList := list.New(items, containersListCustomDelegate{}, width, height)
	servicesList.SetShowHelp(false)
	servicesList.SetShowStatusBar(false)

	servicesList.Title = "Services"
	servicesList.Paginator.ActiveDot = " ● "
	servicesList.Paginator.InactiveDot = " ○ "

	return Model{
		list:        servicesList,
		componentId: 1,
	}
}
