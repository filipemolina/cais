package components

import (
	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/cmds"
	"stack-stitcher/src/constants"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

type GroupDetailsPanelModel struct {
	container   any
	panelWidth  int
	panelHeight int
	isFocused   bool
	componentId int
}

func (m GroupDetailsPanelModel) Init() tea.Cmd {
	return nil
}

func (m GroupDetailsPanelModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		m.container = service
	}

	return m, nil
}

func (m GroupDetailsPanelModel) View() tea.View {
	style := wrapperStyle.
		Width(m.panelWidth).
		Height(m.panelHeight - 1)

	title := appstyles.NormalTitle.Render("Details")

	if m.isFocused {
		style = style.
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(appstyles.PrimaryColor).
			Padding(0, 1)
	}

	if m.container == nil {
		screen := lipgloss.JoinVertical(lipgloss.Left, constants.LOGO, "", "", constants.SLOGAN)

		screen = style.
			Align(lipgloss.Center).
			AlignVertical(lipgloss.Center).
			Render(lipgloss.JoinVertical(lipgloss.Left, title, screen))

		return tea.NewView(screen)
	}

	var basicInfo string

	container, ok := m.container.(types.ServiceConfig)
	if ok {
		basicInfo = BasicInfo(container, m.panelWidth)
	}

	StartButton := Button("Start", "s").View().Content
	StopButton := Button("Stop", "t").View().Content
	RestartButton := Button("Restart", "r").View().Content
	PullButton := Button("Pull", "p").View().Content
	RemoveButton := Button("Remove", "x").View().Content
	leftButtons := lipgloss.NewStyle().
		Width(m.panelWidth - 5).
		AlignHorizontal(lipgloss.Right).
		Render(lipgloss.JoinHorizontal(lipgloss.Left, StartButton, StopButton, RestartButton, PullButton, RemoveButton))

	screen := lipgloss.JoinVertical(lipgloss.Left, title, leftButtons, basicInfo)
	screen = style.Render(screen)

	return tea.NewView(screen)
}

func GroupDetailsPanel(container any) tea.Model {
	service, ok := container.(types.ServiceConfig)

	if ok {
		return GroupDetailsPanelModel{
			container:   service,
			componentId: 2,
		}
	}

	return GroupDetailsPanelModel{
		container:   nil,
		componentId: 2,
	}
}
