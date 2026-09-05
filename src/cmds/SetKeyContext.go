package cmds

import (
	tea "charm.land/bubbletea/v2"

	"github.com/filipemolina/cais/src/keys"
)

// SetKeyContextMsg carries the resolved key context to the footer.
//
// The footer used to derive its own from a dozen mirrored fields, which meant
// two answers to "what is pressable right now" and no mechanism keeping them
// equal. They drifted: the bar went on advertising the action keys over a list
// that had lost its last row, and every one of them was then ignored by the
// panel that had nothing to act on. AppModel resolves the context once and
// hands it down, so there is one answer by construction.
type SetKeyContextMsg keys.Context

func SetKeyContext(ctx keys.Context) func() tea.Msg {
	return func() tea.Msg {
		return SetKeyContextMsg(ctx)
	}
}
