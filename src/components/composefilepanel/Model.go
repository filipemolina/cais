package composefilepanel

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/keys"
)

// ComposeFilePanel replaces the PlaceholderPanel on the Files page. The
// minimal version shows the active compose file's path in the title and a
// read-only, scrollable view of its raw contents. E opens the file in the
// user's editor.
//
// The panel is the sole component on its page, so it fills the whole body
// row (both panel widths plus the gutter). It is not split into a list and
// a details pane - that is a later-phase extension to browse multiple
// compose files.
// The panel is the only component on its page, so it is always active:
// there is no second panel competing for the keyboard, so the E key and
// scrolling always work, matching what the footer advertises.
type Model struct {
	viewport    viewport.Model
	filePath    string
	content     string
	readErr     error
	loaded      bool
	panelWidth  int
	panelHeight int
}

func (m Model) Init() tea.Cmd { return nil }

// New builds the read-only file viewer for the Files page.
func New() tea.Model {
	vp := viewport.New()
	vp.KeyMap = keys.ReadOnlyViewportKeyMap()

	return Model{
		viewport: vp,
	}
}
