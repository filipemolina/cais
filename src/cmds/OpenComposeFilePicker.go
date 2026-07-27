package cmds

import tea "charm.land/bubbletea/v2"

// OpenComposeFilePickerMsg asks AppModel to open the file picker. The panel
// emits this rather than scanning the directory itself: which directory the
// active file lives in is AppModel's business, not a component's.
type OpenComposeFilePickerMsg struct{}

func OpenComposeFilePicker() tea.Cmd {
	return func() tea.Msg { return OpenComposeFilePickerMsg{} }
}
