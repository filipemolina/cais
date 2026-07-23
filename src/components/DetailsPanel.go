package components

import (
	"stack-stitcher/src/cmds"
	"stack-stitcher/src/constants"

	tea "charm.land/bubbletea/v2"
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
	case tea.WindowSizeMsg:
		h, _ := wrapperStyle.GetFrameSize()
		panelWidth := float32(msg.Width - h)
		finalWidth := int(panelWidth * constants.RIGHT_PANEL_WIDTH)

		m.panelWidth = int(finalWidth)
		m.panelHeight = msg.Height - 2

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
		}
	}

	return m, tea.Batch(finalCmds...)
}

func (m DetailsPanelModel) View() tea.View {
	if m.service == nil {
		screen := renderPanelFrame("Details", m.isFocused, m.panelWidth, m.panelHeight, "", "")
		return tea.NewView(screen)
	}

	basicInfo := BasicInfo(*m.service, m.panelWidth)
	buttons := renderActionButtons(m.panelWidth)

	screen := renderPanelFrame("Details", m.isFocused, m.panelWidth, m.panelHeight, basicInfo, buttons)
	return tea.NewView(screen)
}

func DetailsPanel(service *types.ServiceConfig) tea.Model {
	return DetailsPanelModel{
		service:     service,
		componentId: 2,
	}
}
