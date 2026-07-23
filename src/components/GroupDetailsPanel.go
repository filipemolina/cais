package components

import (
	"fmt"
	"slices"
	"stack-stitcher/src/apptypes"
	"stack-stitcher/src/cmds"
	"stack-stitcher/src/constants"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

var groupDetailsPanelActions = map[string]string{
	"s": "start",
	"t": "stop",
	"r": "restart",
	"p": "pull",
	"x": "remove",
}

type GroupDetailsPanelModel struct {
	selectedProfile string
	services        []types.ServiceConfig
	containers      []apptypes.DockerContainer
	panelWidth      int
	panelHeight     int
	isFocused       bool
	componentId     int
}

func (m GroupDetailsPanelModel) Init() tea.Cmd {
	return nil
}

func (m GroupDetailsPanelModel) memberServices() []types.ServiceConfig {
	var members []types.ServiceConfig

	for _, service := range m.services {
		if slices.Contains(service.Profiles, m.selectedProfile) {
			members = append(members, service)
		}
	}

	return members
}

func (m GroupDetailsPanelModel) isServiceRunning(serviceName string) bool {
	for _, container := range m.containers {
		if container.Service == serviceName {
			return container.State == "running"
		}
	}

	return false
}

func (m GroupDetailsPanelModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case cmds.SetSelectedProfileMsg:
		m.selectedProfile = string(msg)

	case cmds.SetServicesListMsg:
		m.services = msg

	case cmds.GetRunningContainersMsg:
		if msg.Err == nil {
			m.containers = msg.Containers
		}

	case tea.KeyPressMsg:
		if m.isFocused && m.selectedProfile != "" {
			if action, ok := groupDetailsPanelActions[msg.String()]; ok {
				actionCmd := cmds.RunDockerAction(action, m.selectedProfile, true)
				finalCmds = append(finalCmds, actionCmd)
			}
		}
	}

	return m, tea.Batch(finalCmds...)
}

func (m GroupDetailsPanelModel) View() tea.View {
	if m.selectedProfile == "" {
		screen := renderPanelFrame("Details", m.isFocused, m.panelWidth, m.panelHeight, "", "")
		return tea.NewView(screen)
	}

	members := m.memberServices()
	runningCount := 0

	memberLines := make([]string, 0, len(members))
	for _, service := range members {
		status := "stopped"
		if m.isServiceRunning(service.Name) {
			status = "running"
			runningCount++
		}

		memberLines = append(memberLines, fmt.Sprintf("%s (%s)", service.Name, status))
	}

	header := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("Profile: %s", m.selectedProfile))
	summary := fmt.Sprintf("%d running, %d stopped", runningCount, len(members)-runningCount)
	body := lipgloss.JoinVertical(lipgloss.Left, append([]string{header, summary, ""}, memberLines...)...)

	buttons := renderActionButtons(m.panelWidth)
	screen := renderPanelFrame("Details", m.isFocused, m.panelWidth, m.panelHeight, body, buttons)

	return tea.NewView(screen)
}

func GroupDetailsPanel() tea.Model {
	return GroupDetailsPanelModel{
		componentId: 2,
	}
}
