package cmds

import (
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/utils"

	tea "charm.land/bubbletea/v2"
)

type NormalizeUngroupedMsg struct {
	Err error
}

// NormalizeUngrouped reconciles the reserved ungrouped profile in fileName
// after a group write while it is materialized: services carrying it
// alongside another profile drop it, and services left with no profile
// regain it. See utils.NormalizeUngrouped.
func NormalizeUngrouped(fileName string) tea.Cmd {
	return func() tea.Msg {
		return NormalizeUngroupedMsg{Err: utils.NormalizeUngrouped(fileName, apptypes.UngroupedGroup)}
	}
}
