package usageoverlay

import (
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/utils"
)

// Model is the usage overlay: shows disk and memory usage as horizontal bars.
type Model struct {
	disk       []utils.DiskUsage
	memTotal   int64
	containers []apptypes.DockerContainer
	spinner    spinner.Model
	loading    bool
	err        error
	termWidth  int
	termHeight int
}

// New builds the usage overlay and dispatches the initial GetDockerUsage command.
// It returns both the model and the command to fetch usage data.
func New(containers []apptypes.DockerContainer) (tea.Model, tea.Cmd) {
	m := Model{
		containers: containers,
		spinner:    chrome.NewSpinner(),
		loading:    true,
	}
	return m, tea.Batch(m.spinner.Tick, cmds.GetDockerUsage(containers))
}

func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}
