package cmds

import (
	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

type SetServicesListMsg []types.ServiceConfig

func SetServicesList(services []types.ServiceConfig) func() tea.Msg {
	return func() tea.Msg {
		return SetServicesListMsg(services)
	}
}
