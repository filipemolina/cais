package keybindingbar

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/cmds"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width

	case cmds.SetFocusMsg:
		m.focusedComponent = int(msg)

	case cmds.SetActivePageMsg:
		m.activePage = string(msg)

	case cmds.SetSelectedGroupMsg:
		m.selectedGroup = string(msg)

	case cmds.SetSelectedServiceMsg:
		m.selectedService = true

	case cmds.SetGroupsListMsg:
		m.groupsListEmpty = len(msg) == 0
		if m.groupsListEmpty {
			m.selectedGroup = ""
		}

	case cmds.SetServicesListMsg:
		m.servicesListEmpty = len(msg) == 0
		if m.servicesListEmpty {
			m.selectedService = false
		}

	case cmds.SetComposeFileMsg:
		m.composeFile = msg.Name
		m.composeFileOthers = len(msg.Others)

	case cmds.SetListFilterStateMsg:
		m.filterState = list.FilterState(msg)

	case cmds.SetEditingStateMsg:
		m.editing = bool(msg)

	case cmds.SetPendingActionMsg:
		m.pendingAction = true

	case cmds.ClearPendingActionMsg:
		m.pendingAction = false
	}
	return m, nil
}
