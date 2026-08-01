package envkeymodal

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// Model represents a modal for entering an env key name (used in add operations).
type Model struct {
	keyInput textinput.Model
	err      error
}

func (m Model) Init() tea.Cmd { return m.keyInput.Focus() }

// New creates a new env key input modal.
func New() tea.Model {
	ti := textinput.New()
	ti.Placeholder = "VARIABLE_NAME"
	ti.CharLimit = 256
	ti.SetWidth(40)

	return Model{
		keyInput: ti,
	}
}

// GetKey returns the entered key.
func (m Model) GetKey() string {
	return m.keyInput.Value()
}

// SetError sets an error message.
func (m *Model) SetError(err error) {
	m.err = err
}
