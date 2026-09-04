package backupslist

import (
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/utils"
)

// rowHeight is how many terminal lines one version row occupies: the
// source/timestamp line, the sha8 line under it, and the row of padding
// above and below them both. It must agree with what renderRow emits - the
// window budgets this many lines per row when working out how many fit, and
// a disagreement either clips the last row or leaves dead space under it.
//
// Four is the same height a services row has, for the same reason: two
// content lines inside a wrapper with Padding(1).
const rowHeight = 4

// rowIndent is how far a row's text sits from the panel's left edge: one
// column for the state bar, one for the wrapper's left padding. The column
// header is indented by the same amount so the labels line up with the
// values.
const rowIndent = 2

// headerHeight is the column header plus the rule under it. Both are pinned
// above the scrolling rows, the way a table header stays put, so scrolling a
// long history never leaves the columns unlabelled.
const headerHeight = 2

// Model is the Backups page's left panel: a merged, newest-first list of
// stored versions of the compose file and the .env, each tagged with the
// source it would restore to.
//
// It owns the cursor and nothing else. The bytes behind the selected row
// belong to backuppreviewpanel, which learns about the selection through
// cmds.SetSelectedBackupMsg rather than by being handed a pointer - the two
// panels are siblings under AppModel, the same as Home's list and details.
type Model struct {
	entries     []utils.BackupEntry
	selectedIdx int
	loading     bool
	empty       bool
	loadErr     error
	panelWidth  int
	panelHeight int
	// rowOffset is the index of the first row on screen. Before it, the
	// panel rendered every entry and let the body clip: a cursor past the
	// visible rows kept moving with nothing on screen to show for it, and
	// G landed on a row that was guaranteed to be invisible.
	rowOffset int
}

func (m Model) Init() tea.Cmd { return nil }

// New builds the version list. It starts empty and loading; the page switch
// issues GetBackups, whose result fills it via BackupListMsg.
func New() tea.Model {
	return Model{loading: true}
}

// selected returns the entry under the cursor, or false when the list is
// empty or the cursor is out of range.
func (m Model) selected() (utils.BackupEntry, bool) {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.entries) {
		return utils.BackupEntry{}, false
	}
	return m.entries[m.selectedIdx], true
}

// publishSelection tells the preview panel which copy the cursor is on. It
// is the one place the list speaks to its sibling, so every path that moves
// the cursor goes through it rather than emitting the message by hand.
func (m Model) publishSelection() tea.Cmd {
	entry, ok := m.selected()
	if !ok {
		return cmds.ClearSelectedBackup()
	}
	return cmds.SetSelectedBackup(entry)
}

// setSize sizes the panel and pulls the cursor back into view, since a
// shorter panel can leave it below a fold that just moved up.
func (m *Model) setSize(width, height int) {
	m.panelWidth = width
	m.panelHeight = height

	m.ensureCursorVisible()
}

// visibleRows is how many whole version rows fit under the pinned header. A
// partial row at the bottom is not counted: a row is a timestamp line and a
// sha line, and half of one is not worth scrolling to.
func (m Model) visibleRows() int {
	usable := chrome.PanelBodyHeight(m.panelHeight) - headerHeight

	return max(1, usable/rowHeight)
}

// ensureCursorVisible scrolls the least amount that brings the selected row
// into view, so moving the cursor one row scrolls one row while a jump to
// either end scrolls straight there.
//
// The window is tracked in whole rows rather than in lines, and only the rows
// inside it are ever rendered. Rendering the whole history into a viewport
// and scrolling that was the obvious alternative and is what the first draft
// of this phase did; it costs a re-render of every row on every cursor move,
// because the selected row is styled differently from the rest. At the
// store's ceiling - MaxBackupsPerSource across two sources - that measured
// 39ms per keystroke, well past a frame. See BenchmarkRenderRowsAtStoreCeiling.
func (m *Model) ensureCursorVisible() {
	visible := m.visibleRows()

	// Clamp first: entries can shrink out from under the offset on a reload.
	maxOffset := max(0, len(m.entries)-visible)
	m.rowOffset = min(m.rowOffset, maxOffset)

	switch {
	case m.selectedIdx < m.rowOffset:
		m.rowOffset = m.selectedIdx
	case m.selectedIdx >= m.rowOffset+visible:
		m.rowOffset = m.selectedIdx - visible + 1
	}

	m.rowOffset = max(0, m.rowOffset)
}

// moveCursorTo puts the cursor on idx, scrolls it into view, and publishes
// the new selection.
func (m *Model) moveCursorTo(idx int) tea.Cmd {
	m.selectedIdx = idx
	m.ensureCursorVisible()

	return m.publishSelection()
}
