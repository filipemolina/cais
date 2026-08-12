package cmds

import tea "charm.land/bubbletea/v2"

// OpenEnvModalMsg opens the env modal: the .env variable table, editor, and
// raw edit, all inside one centered modal. The global v key emits this.
type OpenEnvModalMsg struct{}

// OpenEnvModal opens the env modal.
func OpenEnvModal() tea.Cmd {
	return func() tea.Msg { return OpenEnvModalMsg{} }
}

// RequestRestoreBackupMsg asks AppModel to confirm and then perform a restore
// of the named .bak over its live source file. Source is "compose" or ".env"
// so the model can resolve the live path the copy belongs to.
type RequestRestoreBackupMsg struct {
	Source string
	Name   string
}

// RequestRestoreBackup emits a restore request for the named backup.
func RequestRestoreBackup(source, name string) tea.Cmd {
	return func() tea.Msg { return RequestRestoreBackupMsg{Source: source, Name: name} }
}
