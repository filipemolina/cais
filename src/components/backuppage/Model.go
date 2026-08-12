package backuppage

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/utils"
)

// Model is the Backups page: a merged, newest-first list of stored versions
// of the compose file and the .env (each tagged with its source), beside a
// live preview of the selected copy. It is the sole component on its page, so
// it fills the whole body row and is always focused.
type Model struct {
	entries     []utils.BackupEntry
	selectedIdx int
	// preview holds the selected copy's content. Compose copies are syntax
	// highlighted; .env copies are shown raw.
	preview     string
	previewVP   viewport.Model
	loading     bool
	empty       bool
	loadErr     error
	panelWidth  int
	panelHeight int
	// listWidth and previewWidth split the body row between the two columns.
	listWidth    int
	previewWidth int
}

func (m Model) Init() tea.Cmd { return nil }

// New builds the Backups page panel. The list starts empty; the page switch
// issues GetBackups, whose result fills it via BackupListMsg.
func New() tea.Model {
	vp := viewport.New()
	// The preview is read-only; strip the letter keys so scrolling is arrows
	// and pgup/pgdn only, the same choice composefilepanel makes.
	vp.KeyMap = previewViewportKeyMap()

	return Model{
		previewVP: vp,
		loading:   true,
	}
}

// previewViewportKeyMap returns a viewport keymap with the letter keys
// stripped, so k/j/↑/↓ navigation stays with the list and the preview only
// scrolls when it has focus-worthy overflow. The list owns the arrow keys;
// the preview's own Up/Down keys mirror them so a long copy still scrolls.
func previewViewportKeyMap() viewport.KeyMap {
	unbound := key.NewBinding()

	return viewport.KeyMap{
		PageDown:     key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdn", "page down")),
		PageUp:       key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "page up")),
		HalfPageUp:   key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "½ page up")),
		HalfPageDown: key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "½ page down")),
		Up:           key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:         key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Left:         unbound,
		Right:        unbound,
	}
}

// SelectedPath returns the live file path the selected row would restore to,
// derived from the source tag. It is empty when the list is empty or nothing
// is selected.
func (m Model) SelectedSourceFile() string {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.entries) {
		return ""
	}
	return m.entries[m.selectedIdx].Source
}

// SelectedBackupName returns the .bak name of the selected row, or "".
func (m Model) SelectedBackupName() string {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.entries) {
		return ""
	}
	return m.entries[m.selectedIdx].Name
}

// setSize splits the body row into a list column and a preview column and
// sizes the preview viewport to its half.
func (m *Model) setSize(width, height int) {
	m.panelWidth = width
	m.panelHeight = height

	bodyW := max(1, width-1) // -1 for the accent/selection column
	// Split the body roughly in half, leaving the gutter between the two
	// columns. The list gets the left half, the preview the right.
	half := bodyW / 2
	m.listWidth = max(20, half)
	m.previewWidth = max(20, bodyW-m.listWidth-1)

	m.resizePreview()
}

func (m *Model) resizePreview() {
	frameW, frameH := chrome.WrapperStyle.GetFrameSize()
	// The preview column holds the PanelBody chrome plus a title row.
	m.previewVP.SetWidth(max(1, m.previewWidth-frameW))
	m.previewVP.SetHeight(max(1, m.panelHeight-frameH-2))
}
