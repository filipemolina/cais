package groupdetailspanel

import (
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/components/chrome"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

type Model struct {
	selectedGroup string
	services      []types.ServiceConfig
	containers    []apptypes.DockerContainer
	panelWidth    int
	panelHeight   int
	isFocused     bool
	componentId   int
	pendingAction *chrome.PendingAction
	spinner       spinner.Model
}

func (m Model) Init() tea.Cmd {
	return nil
}

func New() tea.Model {
	return Model{
		componentId: 2,
		spinner:     chrome.NewSpinner(),
	}
}
