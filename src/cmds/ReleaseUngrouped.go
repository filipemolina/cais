package cmds

import (
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/utils"

	tea "charm.land/bubbletea/v2"
)

type ReleaseUngroupedMsg struct {
	Err error
}

// ReleaseUngrouped strips the ungrouped profile from every service carrying
// it in fileName, returning those services to the derived (profile-less)
// state. It is the in-app reversal of AdoptUngrouped.
func ReleaseUngrouped(fileName string) tea.Cmd {
	return func() tea.Msg {
		return ReleaseUngroupedMsg{Err: utils.RemoveGroupTag(fileName, apptypes.UngroupedGroup)}
	}
}
