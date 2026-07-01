package components

import (
	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/apptypes"
	"stack-stitcher/src/cmds"
	"stack-stitcher/src/constants"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var wrapperStyle = lipgloss.NewStyle().
	Padding(2)

var logoStyle = lipgloss.NewStyle().
	Align(lipgloss.Center)

type DetailsPanelModel struct {
	container   *apptypes.DockerContainer
	panelWidth  int
	panelHeight int
	isFocused   bool
	componentId int
}

func (m DetailsPanelModel) Init() tea.Cmd {
	return nil
}

func (m DetailsPanelModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	}
	return m, nil
}

func (m DetailsPanelModel) View() tea.View {
	style := wrapperStyle
	screen := lipgloss.JoinVertical(lipgloss.Left, constants.LOGO, "", "", constants.SLOGAN)

	if m.isFocused {
		style = style.
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(appstyles.PrimaryColor).
			Padding(1)
	}

	title := appstyles.NormalTitle.Render("Details")

	screen = style.
		Width(m.panelWidth).
		Height(m.panelHeight - 1).
		Align(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, screen))

	return tea.NewView(screen)
}

func DetailsPanel(container *apptypes.DockerContainer) tea.Model {
	return DetailsPanelModel{
		container:   container,
		componentId: 2,
	}
}
