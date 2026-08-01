package envkeymodal

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// Model represents a modal for entering/editing an env variable (key and value).
type Model struct {
	keyInput   textinput.Model
	valueInput textinput.Model
	focused    int // 0 for key, 1 for value
	isEdit     bool
	err        error
}

func (m Model) Init() tea.Cmd {
	if m.focused == 0 {
		return m.keyInput.Focus()
	}
	return m.valueInput.Focus()
}

// New creates a new env variable input modal for adding a new variable.
func New() tea.Model {
	keyInput := textinput.New()
	keyInput.Placeholder = "VARIABLE_NAME"
	keyInput.CharLimit = 256
	keyInput.SetWidth(40)

	valueInput := textinput.New()
	valueInput.Placeholder = "value"
	valueInput.CharLimit = 4096
	valueInput.SetWidth(40)
	valueInput.EchoMode = textinput.EchoPassword

	return Model{
		keyInput:   keyInput,
		valueInput: valueInput,
		focused:    0,
		isEdit:     false,
	}
}

// NewForEdit creates a new env variable modal for editing an existing variable.
func NewForEdit(key, value string) tea.Model {
	keyInput := textinput.New()
	keyInput.Placeholder = "VARIABLE_NAME"
	keyInput.CharLimit = 256
	keyInput.SetWidth(40)
	keyInput.SetValue(key)
	keyInput.Blur()

	valueInput := textinput.New()
	valueInput.Placeholder = "value"
	valueInput.CharLimit = 4096
	valueInput.SetWidth(40)
	valueInput.SetValue(value)
	valueInput.EchoMode = textinput.EchoPassword

	return Model{
		keyInput:   keyInput,
		valueInput: valueInput,
		focused:    1,
		isEdit:     true,
	}
}

// GetKey returns the entered key.
func (m Model) GetKey() string {
	return m.keyInput.Value()
}

// GetValue returns the entered value.
func (m Model) GetValue() string {
	return m.valueInput.Value()
}

// SetError sets an error message.
func (m *Model) SetError(err error) {
	m.err = err
}
