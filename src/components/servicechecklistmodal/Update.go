package servicechecklistmodal

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var finalCmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(keyMsg, keys.Overlay.Cancel):
			return m, cmds.CloseModal(nil)

		case key.Matches(keyMsg, keys.Overlay.Toggle):
			index := m.list.GlobalIndex()
			if item, ok := m.list.SelectedItem().(apptypes.CheckableServiceItem); ok {
				item.Checked = !item.Checked
				finalCmds = append(finalCmds, m.list.SetItem(index, item))
			}

		case key.Matches(keyMsg, keys.Overlay.Submit):
			checked := m.CheckedNames()
			if m.isEdit {
				// Editing an existing group allows empty membership:
				// unchecking every service removes the group from the
				// list, which is the same outcome as deleting it.
				return m, cmds.CloseModal(cmds.RequestEditGroup(m.groupName, checked))
			}
			if len(checked) > 0 {
				return m, cmds.CloseModal(cmds.RequestCreateGroup(m.groupName, checked))
			}
		}
	}

	var listCmd tea.Cmd
	m.list, listCmd = m.list.Update(msg)
	finalCmds = append(finalCmds, listCmd)

	return m, tea.Batch(finalCmds...)
}
