package backuppreviewpanel

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/keys"
	"github.com/filipemolina/cais/src/utils"
)

// Model is the Backups page's right panel: the contents of whichever stored
// copy the list's cursor is on.
//
// It does not know the list exists. The selection arrives as
// cmds.SetSelectedBackupMsg and the bytes are read here, so the panel that
// owns the viewport is also the one that owns the read - the list stays a
// cursor over metadata.
//
// Compose copies are syntax highlighted; .env copies are shown raw. That is
// deliberate and is the page's standing contract: the preview is the exact
// bytes a restore would write, secrets included. See docs/DESIGN.md.
type Model struct {
	// entry is the copy being shown. The zero value means nothing is
	// selected, which is the empty state.
	entry   utils.BackupEntry
	content string
	loadErr error

	vp          viewport.Model
	panelWidth  int
	panelHeight int
}

func (m Model) Init() tea.Cmd { return nil }

// New builds the preview panel. It starts with nothing selected; the list
// publishes a selection as soon as the store has been read.
func New() tea.Model {
	vp := viewport.New()
	// The preview is read-only, so it takes the shared read-only map. No key
	// reaches it yet: the panel has no focus to hold, so Update does not
	// route keys here at all. Focus lands in the next phase - see
	// docs/plans/backups-rework.md.
	vp.KeyMap = keys.ReadOnlyViewportKeyMap()

	return Model{vp: vp}
}

// hasSelection reports whether a copy is being shown. An empty Name is the
// cleared selection cmds.ClearSelectedBackup sends.
func (m Model) hasSelection() bool {
	return m.entry.Name != ""
}

func (m *Model) setSize(width, height int) {
	m.panelWidth = width
	m.panelHeight = height

	frameW, frameH := chrome.WrapperStyle.GetFrameSize()
	// The panel holds the PanelBody chrome plus the header row naming the
	// source and SHA-8.
	m.vp.SetWidth(max(1, m.panelWidth-frameW))
	m.vp.SetHeight(max(1, m.panelHeight-frameH-2))
}
