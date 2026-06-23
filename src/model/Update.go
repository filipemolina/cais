package model

import tea "charm.land/bubbletea/v2"

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// Handle keyboard events
	case tea.KeyPressMsg:

		switch msg.String() {

		// Quit the program on Ctrl+c or q
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	return m, nil
}
