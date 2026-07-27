package cmds

import tea "charm.land/bubbletea/v2"

// SwitchComposeFileMsg asks AppModel to make path the active compose file.
// The picker modal emits this as its follow-up; AppModel points the source
// at path and reloads, which is exactly what passing --file at startup does.
type SwitchComposeFileMsg struct {
	Path string
}

func SwitchComposeFile(path string) tea.Cmd {
	return func() tea.Msg { return SwitchComposeFileMsg{Path: path} }
}
