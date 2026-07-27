package cmds

import (
	"github.com/filipemolina/stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

type EditGroupMsg struct {
	Err error
}

// EditGroupRequestMsg asks AppModel to reconcile a group's membership.
// The checklist modal emits this instead of the command itself: like the
// details panels, it has no business knowing which compose file is loaded.
type EditGroupRequestMsg struct {
	GroupName string
	Members   []string
}

// RequestEditGroup asks AppModel to reconcile groupName to exactly members.
func RequestEditGroup(groupName string, members []string) tea.Cmd {
	return func() tea.Msg {
		return EditGroupRequestMsg{GroupName: groupName, Members: members}
	}
}

// EditGroup reconciles groupName's membership in fileName, the compose
// file AppModel has loaded. Services in members that lack the tag get it
// added; services not in members that carry it get it removed.
func EditGroup(fileName string, groupName string, members []string) tea.Cmd {
	return func() tea.Msg {
		return EditGroupMsg{Err: utils.SetGroupMembers(fileName, groupName, members)}
	}
}
