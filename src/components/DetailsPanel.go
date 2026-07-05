package components

import (
	"encoding/json"
	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/cmds"
	"stack-stitcher/src/constants"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

var wrapperStyle = lipgloss.NewStyle().
	Padding(1, 2)

var logoStyle = lipgloss.NewStyle().
	Align(lipgloss.Center)

type DetailsPanelModel struct {
	container   any
	panelWidth  int
	panelHeight int
	isFocused   bool
	componentId int
}

func (m DetailsPanelModel) Init() tea.Cmd {
	return nil
}

func (m DetailsPanelModel) Update(msg tea.Msg) (DetailsPanelModel, tea.Cmd) {
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

func (m DetailsPanelModel) View() tea.View {
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

	containerPrint, err := json.MarshalIndent(m.container, "", " ")
	if err != nil {
		panic(err)
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

	screen := lipgloss.JoinVertical(lipgloss.Left, title, leftButtons, string(containerPrint))
	screen = style.Render(screen)

	return tea.NewView(screen)
}

func DetailsPanel(container any) DetailsPanelModel {
	service, ok := container.(types.ServiceConfig)

	if ok {
		return DetailsPanelModel{
			container:   service,
			componentId: 2,
		}
	}

	return DetailsPanelModel{
		container:   nil,
		componentId: 2,
	}
}
