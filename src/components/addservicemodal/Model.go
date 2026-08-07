package addservicemodal

import (
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/components/servicefieldsstep"

	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
)

type Model struct {
	fileName             string
	existingServiceNames []string
	step                 servicefieldsstep.Model
	initCmd              tea.Cmd
}

func (m Model) Init() tea.Cmd {
	return m.initCmd
}

func New(fileName string, existingServiceNames []string) tea.Model {
	s, cmd := servicefieldsstep.New("New service", func(name, image string) tea.Cmd {
		return cmds.AddService(fileName, name, image)
	})
	return Model{
		fileName:             fileName,
		existingServiceNames: existingServiceNames,
		step:                 s,
		initCmd:              cmd,
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := m.step.Update(msg)
	m.step = updated
	return m, cmd
}

func (m Model) View() tea.View {
	return tea.NewView(chrome.ModalSurface(
		appstyles.Active.ModalBg,
		m.step.View(),
	))
}
