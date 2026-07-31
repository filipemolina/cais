package helpoverlay

import (
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

// helpOverlayMaxWidth caps the content column so hint runs wrap in a few
// places on wide terminals rather than stretching into one unreadable line.
const helpOverlayMaxWidth = 64

// Model is the ? overlay: every key in the app, grouped by scope
// and rendered from keys.Catalog, so what it says is what the handlers do.
// Rows the user could not press in the screen it was opened from are dimmed.
// It also names the compose-file candidates that lost to the loaded one - the
// footer only has room to count them.
type Model struct {
	catalog      []keys.Scope
	composeFiles []string
	termWidth    int
}

func (m Model) Init() tea.Cmd { return nil }

// New builds the help overlay for the screen described by ctx (which
// keys are pressable), the compose-file candidates in priority order, and the
// terminal width for wrapping.
func New(ctx keys.Context, composeFiles []string, termWidth int) tea.Model {
	return Model{
		catalog:      keys.Catalog(ctx),
		composeFiles: composeFiles,
		termWidth:    termWidth,
	}
}
