package envpanel

import (
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/cmds"
)

// Model represents the Env page panel showing .env variables.
type Model struct {
	envPath         string
	entries         []cmds.EnvEntry
	loading         bool
	loadErr         error
	panelWidth      int
	panelHeight     int
	selectedIdx     int
	revealedIdx     int // Index of currently revealed row (-1 if none)
	parseErrorCount int
}

func (m Model) Init() tea.Cmd { return nil }

// New creates a new EnvPanel.
func New() tea.Model {
	return Model{
		revealedIdx: -1,
	}
}

// SetSize updates the panel dimensions.
func (m *Model) SetSize(width, height int) {
	m.panelWidth = width
	m.panelHeight = height
}

// SetEnvEntries updates the env entries to display.
func (m *Model) SetEnvEntries(path string, entries []cmds.EnvEntry, parseErrors int) {
	m.envPath = path
	m.entries = entries
	m.parseErrorCount = parseErrors
	m.loading = false
	m.selectedIdx = 0
	m.revealedIdx = -1
}

// SetLoadError sets an error state.
func (m *Model) SetLoadError(err error) {
	m.loadErr = err
	m.loading = false
}

// SetLoading sets the loading state.
func (m *Model) SetLoading(loading bool) {
	m.loading = loading
}

// GetSelectedKey returns the key of the currently selected env entry, or empty string.
func (m *Model) GetSelectedKey() string {
	if m.selectedIdx >= 0 && m.selectedIdx < len(m.entries) {
		return m.entries[m.selectedIdx].Key
	}
	return ""
}

// GetSelectedValue returns the value of the currently selected entry.
func (m *Model) GetSelectedValue() string {
	if m.selectedIdx >= 0 && m.selectedIdx < len(m.entries) {
		return m.entries[m.selectedIdx].Value
	}
	return ""
}

// Reveal sets the currently revealed index.
func (m *Model) Reveal(idx int) {
	m.revealedIdx = idx
}

// ClearReveal hides the revealed value.
func (m *Model) ClearReveal() {
	m.revealedIdx = -1
}

// IsRevealed checks if a specific index is revealed.
func (m *Model) IsRevealed(idx int) bool {
	return m.revealedIdx == idx
}
