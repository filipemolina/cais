package envmodal

import (
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/cmds"
)

// Model is the env experience as a centered modal: a key/value table of the
// .env file, with reveal, copy, add, edit, delete, an external-editor open,
// and a raw edit, all inside one surface. It replaces the old Env page.
type Model struct {
	envPath     string
	entries     []cmds.EnvEntry
	loading     bool
	loadErr     error
	selectedIdx int
	// revealedIdx is the row whose value is shown in clear text (-1 none).
	revealedIdx int
	// termHeight is the terminal height in rows, used to size the modal.
	termHeight int
	// parseErrorCount counts lines the parser flagged (shown in the header).
	parseErrorCount int
}

func (m Model) Init() tea.Cmd { return nil }

// New builds the env modal for envPath. It does not load on its own: the
// caller (AppModel, on OpenEnvModalMsg) issues GetEnvFileContents so the
// table fills, the same way the page used to.
func New(envPath string, termHeight int) tea.Model {
	return Model{
		envPath:     envPath,
		revealedIdx: -1,
		loading:     true,
		termHeight:  termHeight,
	}
}

// EnvPath returns the .env path the modal is showing.
func (m Model) EnvPath() string { return m.envPath }

// SetEntries fills the table from a parse result.
func (m *Model) SetEntries(path string, entries []cmds.EnvEntry, parseErrors int) {
	m.envPath = path
	m.entries = entries
	m.parseErrorCount = parseErrors
	m.loading = false
	m.loadErr = nil
	m.selectedIdx = 0
	m.revealedIdx = -1
}

// SetLoadError records a read failure.
func (m *Model) SetLoadError(err error) {
	m.loadErr = err
	m.loading = false
}

// selectedVar returns the highlighted real variable (source == "var"), or nil
// when the cursor is on a comment/blank/parse-error row or the list is empty.
func (m *Model) selectedVar() *cmds.EnvEntry {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.entries) {
		return nil
	}
	entry := &m.entries[m.selectedIdx]
	if entry.Source != "var" {
		return nil
	}
	return entry
}
