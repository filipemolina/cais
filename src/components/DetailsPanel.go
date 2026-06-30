package components

import (
	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/apptypes"
	"stack-stitcher/src/constants"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var wrapperStyle = lipgloss.NewStyle()

// Border(lipgloss.NormalBorder()).
// BorderForeground(appstyles.PrimaryColor).
// Padding(1)

var logoStyle = lipgloss.NewStyle().
	Align(lipgloss.Center)

type DetailsPanelModel struct {
	container   *apptypes.DockerContainer
	panelWidth  int
	panelHeight int
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
	}
	return m, nil
}

func (m DetailsPanelModel) View() tea.View {
	screen := lipgloss.JoinVertical(lipgloss.Left, constants.LOGO, "", "", constants.SLOGAN)

	screen = wrapperStyle.
		Width(m.panelWidth).
		Height(m.panelHeight - 1).
		Align(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Render(screen)

	title := appstyles.NormalTitle.Render("Details")

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, title, screen))
}

func DetailsPanel(container *apptypes.DockerContainer) tea.Model {
	return DetailsPanelModel{
		container: container,
	}
}
