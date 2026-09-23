package groupdetailspanel

import (
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/utils"

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
	pendingAction *chrome.PendingAction
	spinner       spinner.Model

	// imageUpdates is the image update check's answer, keyed by service
	// name. It is what puts the update glyph in a member's NAME cell and
	// what the updates table below the member table is built from. Only a
	// Stale entry renders as anything (chrome.UpdateGlyph for why).
	imageUpdates map[string]utils.ImageUpdate
}

func (m Model) Init() tea.Cmd {
	return nil
}

func New() tea.Model {
	return Model{
		spinner: chrome.NewSpinner(),
	}
}
