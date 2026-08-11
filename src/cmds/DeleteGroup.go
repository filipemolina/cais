package cmds

import (
	"github.com/filipemolina/cais/src/utils"

	tea "charm.land/bubbletea/v2"
)

type DeleteGroupMsg struct {
	Err error
}

// DeleteGroup removes a group tag from every service that carries it in
// fileName, which is the compose file AppModel has loaded. Re-resolving it
// here would write to whichever file the current directory happens to offer,
// which stops being the loaded one the moment --file points elsewhere.
func DeleteGroup(fileName string, name string) tea.Cmd {
	return func() tea.Msg {
		return DeleteGroupMsg{Err: utils.RemoveGroupTag(fileName, name)}
	}
}
