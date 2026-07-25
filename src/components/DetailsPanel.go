package components

import (
	"stack-stitcher/src/cmds"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

var detailsPanelActions = map[string]string{
	"s": "start",
	"t": "stop",
	"r": "restart",
	"p": "pull",
	"x": "remove",
}

type DetailsPanelModel struct {
	service     *types.ServiceConfig
	panelWidth  int
	panelHeight int
	isFocused   bool
	componentId int
}

func (m DetailsPanelModel) Init() tea.Cmd {
	return nil
}

func (m DetailsPanelModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var finalCmds []tea.Cmd

	switch msg := msg.(type) {
	// Both dimensions come from AppModel. Deriving them from WindowSizeMsg
	// here would leave the panel at width 0 whenever the Dashboard wasn't
	// the active page at resize time.
	case cmds.SetBodyLayoutMsg:
		m.panelWidth = msg.RightWidth
		m.panelHeight = msg.Height

	case cmds.SetFocusMsg:
		if int(msg) == m.componentId {
			m.isFocused = true
		} else {
			m.isFocused = false
		}

	case cmds.SetSelectedServiceMsg:
		service := types.ServiceConfig(msg)
		m.service = &service

	case tea.KeyPressMsg:
		if m.isFocused && m.service != nil {
			if action, ok := detailsPanelActions[msg.String()]; ok {
				actionCmd := cmds.RunDockerAction(action, m.service.Name, false)
				finalCmds = append(finalCmds, actionCmd)
			}

			if msg.String() == "l" {
				finalCmds = append(finalCmds, cmds.OpenLogsModal(m.service.Name, false))
			}
		}
	}

	return m, tea.Batch(finalCmds...)
}

func (m DetailsPanelModel) View() tea.View {
	bodyWidth := max(1, panelBodyWidth(m.panelWidth))
	bodyAvail := max(1, panelBodyHeight(m.panelHeight))

	if m.service == nil {
		body := renderEmptyCard(bodyWidth, bodyAvail, "Select a service",
			"Pick a service from the list to see its details.",
			"↑/↓", "then space")
		screen := renderPanelFrame("Details", m.isFocused, m.panelWidth, m.panelHeight, body)
		return tea.NewView(screen)
	}

	basicInfo := BasicInfo(*m.service, bodyWidth)
	buttons := renderActionButtons(bodyWidth)

	body := lipgloss.JoinVertical(lipgloss.Left, basicInfo, buttons)
	body = lipgloss.NewStyle().MaxHeight(bodyAvail).Render(body)

	screen := renderPanelFrame("Details", m.isFocused, m.panelWidth, m.panelHeight, body)
	return tea.NewView(screen)
}

func DetailsPanel(service *types.ServiceConfig) tea.Model {
	return DetailsPanelModel{
		service:     service,
		componentId: 2,
	}
}
