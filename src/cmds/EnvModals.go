package cmds

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
