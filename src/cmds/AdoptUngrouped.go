package cmds

import (
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/utils"

	tea "charm.land/bubbletea/v2"
)

type AdoptUngroupedMsg struct {
	Err error
}

// AdoptUngrouped writes profiles: [ungrouped] onto every named service in
// fileName. This is the opt-in materialization of the derived ungrouped row:
// the caller has already shown the confirmation that spells out the docker
// compose up consequence.
func AdoptUngrouped(fileName string, services []string) tea.Cmd {
	return func() tea.Msg {
		return AdoptUngroupedMsg{Err: utils.AddGroupTag(fileName, apptypes.UngroupedGroup, services)}
	}
}
