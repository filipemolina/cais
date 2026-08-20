package cmds

import (
	"github.com/filipemolina/cais/src/utils"

	tea "charm.land/bubbletea/v2"
)

type DeleteServiceMsg struct {
	Err error
}

// DeleteService removes a service's entry from fileName, which is the
// compose file AppModel has loaded. Same split as DeleteGroup: re-resolving
// the file here would write to whichever one the current directory happens
// to offer, which stops being the loaded one the moment --file points
// elsewhere.
func DeleteService(fileName string, serviceName string) tea.Cmd {
	return func() tea.Msg {
		return DeleteServiceMsg{Err: utils.DeleteService(fileName, serviceName)}
	}
}
