package cmds

import (
	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

type SetSelectedServiceMsg types.ServiceConfig

func SetSelectedService(service types.ServiceConfig) func() tea.Msg {
	return func() tea.Msg {
		return SetSelectedServiceMsg(service)
	}
}
