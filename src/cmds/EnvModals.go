package cmds

import tea "charm.land/bubbletea/v2"

// OpenEnvKeyModalMsg requests opening the modal for adding a new env variable.
type OpenEnvKeyModalMsg struct{}

// OpenEnvEditModalMsg requests opening the modal for editing an env variable value.
type OpenEnvEditModalMsg struct {
	Key   string
	Value string
}

// OpenEnvDeleteConfirmMsg requests opening the confirm modal for deleting a variable.
type OpenEnvDeleteConfirmMsg struct {
	Key string
}

// OpenEnvRawEditMsg requests opening the raw .env file editor (textarea).
type OpenEnvRawEditMsg struct{}

// RequestAddEnvVariable asks AppModel to add or update an env variable.
func RequestAddEnvVariable(key, value string) tea.Cmd {
	return func() tea.Msg {
		return AddEnvVariableRequestMsg{Key: key, Value: value}
	}
}

type AddEnvVariableRequestMsg struct {
	Key   string
	Value string
}

// RequestEditEnvVariable asks AppModel to update an env variable value.
func RequestEditEnvVariable(key, value string) tea.Cmd {
	return func() tea.Msg {
		return EditEnvVariableRequestMsg{Key: key, Value: value}
	}
}

type EditEnvVariableRequestMsg struct {
	Key   string
	Value string
}

// RequestDeleteEnvVariable asks AppModel to delete an env variable.
func RequestDeleteEnvVariable(key string) tea.Cmd {
	return func() tea.Msg {
		return DeleteEnvVariableRequestMsg{Key: key}
	}
}

type DeleteEnvVariableRequestMsg struct {
	Key string
}
