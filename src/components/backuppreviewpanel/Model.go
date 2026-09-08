package backuppreviewpanel

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/diff"
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
	// read is whether the current entry's read has landed. It distinguishes
	// "nothing read yet" from "the copy is empty", which the diff needs: an
	// empty copy against a non-empty live file is a real diff - restoring
	// it would empty the file - not an absence of one.
	read bool

	// live is each source's current on-disk state, from BackupListMsg. The
	// panel does not know which files are loaded - AppModel does, and
	// resolves the paths - so the bytes arrive on the message rather than
	// being read here. Keyed by the same Source label the entries carry.
	live map[string]utils.LiveSource
	// lines is the whole-file diff of the selected copy against its live
	// file, rebuilt by recomputeDiff whenever either side changes. nil
	// means "not available" - no live bytes, or the read has not landed -
	// which the rendering phase answers by showing the copy's plain bytes.
	// The rendering itself is Phase 5; until then the viewport keeps
	// showing the copy exactly as before.
	lines []diff.Line

	vp          viewport.Model
	panelWidth  int
	panelHeight int
	// focused is whether the arrows are driving this panel's viewport rather
	// than the list's cursor. AppModel owns the answer and broadcasts it; this
	// is the panel's copy. It starts false: the page opens on the list, and
	// there is nothing here to scroll until the cursor picks a row.
	focused bool
}

func (m Model) Init() tea.Cmd { return nil }

// New builds the preview panel. It starts with nothing selected; the list
// publishes a selection as soon as the store has been read.
func New() tea.Model {
	vp := viewport.New()
	// The preview is read-only, so it takes the shared read-only map: the
	// arrows, pgup/pgdn and the ctrl half-pages, with the vim letters and
	// horizontal scrolling dropped. Update routes keys here only while this
	// panel holds focus, so the map is live for exactly half the page.
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
