package cmds

import (
	"github.com/filipemolina/stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

// AddHealthcheckMsg reports the result of inserting or replacing a
// service's healthcheck: block.
type AddHealthcheckMsg struct {
	ServiceName string
	Err         error
}

// AddHealthcheckRequestMsg asks AppModel to apply template to serviceName.
// The picker emits this instead of the command itself: like every other
// write-path modal, it has no business knowing which compose file is
// loaded.
type AddHealthcheckRequestMsg struct {
	ServiceName string
	Template    utils.HealthcheckTemplate
	Port        string
}

// RequestAddHealthcheck asks AppModel to insert template into serviceName's
// compose entry. port is only meaningful for the generic template.
func RequestAddHealthcheck(serviceName string, template utils.HealthcheckTemplate, port string) tea.Cmd {
	return func() tea.Msg {
		return AddHealthcheckRequestMsg{ServiceName: serviceName, Template: template, Port: port}
	}
}

// AddHealthcheck applies template to serviceName in the compose file at
// fileName, the file AppModel has loaded.
func AddHealthcheck(fileName, serviceName string, template utils.HealthcheckTemplate, port string) tea.Cmd {
	return func() tea.Msg {
		return AddHealthcheckMsg{
			ServiceName: serviceName,
			Err:         utils.ApplyHealthcheck(fileName, serviceName, template, port),
		}
	}
}
