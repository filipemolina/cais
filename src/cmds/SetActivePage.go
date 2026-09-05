package cmds

import (
	tea "charm.land/bubbletea/v2"

	"github.com/filipemolina/cais/src/apptypes"
)

type SetActivePageMsg apptypes.Page

func SetActivePage(pageTitle apptypes.Page) func() tea.Msg {
	return func() tea.Msg {
		return SetActivePageMsg(pageTitle)
	}
}
