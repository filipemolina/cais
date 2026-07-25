package cmds

import (
	"stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

// OpenEditorMsg asks AppModel to hand the terminal to the user's editor.
// Panels emit this rather than the command itself: which file is open is
// config state that belongs to AppModel, not to a panel.
type OpenEditorMsg struct{}

// EditorClosedMsg reports that the editor exited and the app has the
// terminal back. Err is set if the editor could not be started or exited
// non-zero.
type EditorClosedMsg struct {
	Err error
}

// OpenEditor asks AppModel to open the compose file for editing.
func OpenEditor() tea.Cmd {
	return func() tea.Msg {
		return OpenEditorMsg{}
	}
}

// RunEditor suspends the TUI, gives the terminal to the user's editor, and
// resumes when it exits. Bubble Tea handles the handover: it releases the
// terminal, restores raw mode and the alternate screen afterwards, and
// delivers the callback's message through the normal update loop.
//
// Nothing here inspects or writes the file. The user is editing the real
// compose file directly, exactly as they would outside the app, so there is
// nothing to validate and nothing to roll back - a broken save surfaces as
// an ordinary config-load error on the reload that follows.
func RunEditor(path string) tea.Cmd {
	return tea.ExecProcess(utils.EditorCommand(path), func(err error) tea.Msg {
		return EditorClosedMsg{Err: err}
	})
}
