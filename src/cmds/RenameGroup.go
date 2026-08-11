package cmds

import (
	"github.com/filipemolina/cais/src/utils"

	tea "charm.land/bubbletea/v2"
)

type RenameGroupMsg struct {
	Err error
	// NewName rides the result so AppModel can keep the renamed group
	// selected after the reload - selection.groupName still holds the old
	// name until then.
	NewName string
}

// RenameGroupRequestMsg asks AppModel to rename GroupName to NewName in
// the loaded compose file. The rename modal emits this instead of the
// command itself, the same split as CreateGroupRequestMsg.
type RenameGroupRequestMsg struct {
	GroupName string
	NewName   string
}

// RequestRenameGroup asks AppModel to rename groupName to newName.
func RequestRenameGroup(groupName string, newName string) tea.Cmd {
	return func() tea.Msg {
		return RenameGroupRequestMsg{GroupName: groupName, NewName: newName}
	}
}

// RenameGroup renames groupName to newName in fileName, the compose file
// AppModel has loaded. See utils.RenameGroupTag.
func RenameGroup(fileName string, groupName string, newName string) tea.Cmd {
	return func() tea.Msg {
		return RenameGroupMsg{Err: utils.RenameGroupTag(fileName, groupName, newName), NewName: newName}
	}
}
