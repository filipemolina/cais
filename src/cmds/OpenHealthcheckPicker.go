package cmds

import tea "charm.land/bubbletea/v2"

// OpenHealthcheckPickerMsg asks AppModel to open the healthcheck template
// picker for serviceName - h on the Services details panel.
type OpenHealthcheckPickerMsg struct {
	ServiceName string
}

// OpenHealthcheckPicker opens the picker for serviceName.
func OpenHealthcheckPicker(serviceName string) tea.Cmd {
	return func() tea.Msg {
		return OpenHealthcheckPickerMsg{ServiceName: serviceName}
	}
}
