package cmds

import tea "charm.land/bubbletea/v2"

// OpenAddServiceModalMsg asks AppModel to open the "add a service" modal -
// n on the Services page. Same path every other modal takes.
type OpenAddServiceModalMsg struct{}

// OpenAddServiceModal opens the modal that collects a new service's name and
// image before handing off to the inline editor.
func OpenAddServiceModal() tea.Cmd {
	return func() tea.Msg { return OpenAddServiceModalMsg{} }
}
