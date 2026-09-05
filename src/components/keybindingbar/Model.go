package keybindingbar

import (
	tea "charm.land/bubbletea/v2"

	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/keys"
)

// KeybindingBar is a single-line footer that shows which compose file the app
// resolved and the keys available right now.
//
// It does not decide what is available. AppModel resolves one keys.Context per
// frame and sends it down as cmds.SetKeyContextMsg; the bar renders that and
// nothing else. It used to mirror a dozen pieces of screen state and rebuild
// the context itself, which is a second answer to a question that has one
// right answer - and the two drifted.
type Model struct {
	// keyContext is what the footer renders from, resolved by AppModel and
	// handed down whole. The bar deliberately keeps no state of its own that
	// feeds it: mirroring the pieces and re-deriving the answer here is what
	// let the footer and the panels disagree about what was pressable.
	keyContext    keys.Context
	terminalWidth int
	composeFile   string
	// composeFileOthers is how many candidates lost to composeFile. The
	// winner is the whole story only when it is the only one, so a +N marks
	// the rest; the help overlay names them.
	//
	// This and composeFile stay the bar's own: they are the other half of the
	// footer's job, and no key depends on them.
	composeFileOthers int
}

func (m Model) Init() tea.Cmd { return nil }

// New builds the footer keybinding bar.
func New() tea.Model {
	// The empty-list defaults are the honest starting point: nothing has been
	// loaded yet, so no list has rows and no selection-dependent verb is
	// offered. AppModel replaces the whole context on the first Update.
	return Model{
		keyContext: keys.Context{Page: apptypes.PageHome, ListEmpty: true},
	}
}
