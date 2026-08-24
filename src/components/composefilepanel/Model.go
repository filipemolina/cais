package composefilepanel

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
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

// composeFileViewportKeyMap returns a viewport keymap with the letter keys
// stripped. The viewport's DefaultKeyMap claims f, b, u, d, h, l, k, j -
// the same collisions that forced the list keymap work. A read-only file
// viewer has no use for horizontal scrolling or vim-style half-pages, so
// the letter keys are unbound and only the arrows and pgup/pgdn remain.
func composeFileViewportKeyMap() viewport.KeyMap {
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

// New builds the read-only file viewer for the Files page.
func New() tea.Model {
	vp := viewport.New()
	vp.KeyMap = composeFileViewportKeyMap()

	return Model{
		viewport: vp,
	}
}
