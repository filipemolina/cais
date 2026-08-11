package envkeymodal

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/keys"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keys.Overlay.Cancel):
			return m, cmds.CloseModal(nil)

		case key.Matches(msg, keys.Overlay.NextField):
			m.focused = (m.focused + 1) % 2
			if m.focused == 0 {
				m.keyInput.Focus()
				m.valueInput.Blur()
			} else {
				m.keyInput.Blur()
				m.valueInput.Focus()
			}
			return m, nil

		case key.Matches(msg, keys.Overlay.Submit):
			if m.keyInput.Value() == "" {
				m.SetError(fmt.Errorf("variable name is required"))
				return m, nil
			}

			var cmd tea.Cmd
			if m.isEdit {
				cmd = cmds.CloseModal(cmds.RequestEditEnvVariable(m.keyInput.Value(), m.valueInput.Value()))
			} else {
				cmd = cmds.CloseModal(cmds.RequestAddEnvVariable(m.keyInput.Value(), m.valueInput.Value()))
			}
			return m, cmd

		default:
			var cmd tea.Cmd
			if m.focused == 0 {
				m.keyInput, cmd = m.keyInput.Update(msg)
			} else {
				m.valueInput, cmd = m.valueInput.Update(msg)
			}
			return m, cmd
		}
	}
	return m, nil
}
