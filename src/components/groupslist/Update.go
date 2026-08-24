package groupslist

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/keys"
)

// syncActiveIndex points the delegate at the row holding activeGroup, or at
// no row at all when it isn't in the list. Runs on both the list and the
// selection changing, since they arrive as unordered separate messages.
func (m *Model) syncActiveIndex() {
	active := -1

	for i, item := range m.list.Items() {
		if group, ok := item.(apptypes.GroupListItem); ok && group.Name == m.activeGroup {
			active = i
			break
		}
	}

	m.listDelegate.activeIndex = active
	m.list.SetDelegate(m.listDelegate)
}

// resizeList sizes the inner list to the space left inside the panel box
// after the wrapper padding and the stats footer. Called whenever either the
// box or the footer changes.
func (m *Model) resizeList() {
	h, v := chrome.ListWrapperStyle.GetFrameSize()

	m.list.SetSize(
		max(0, m.panelWidth-h),
		max(0, m.panelHeight-v-m.footerHeight()),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var finalCmds []tea.Cmd

	// The footer advertises different keys depending on this, so the transition
	// has to be announced. Taken before the inner list sees the message.
	filterStateBefore := m.list.FilterState()

	switch msg := msg.(type) {
	// The panel's box comes from AppModel; the inner list is sized to what
	// is left inside the wrapper's padding.
	case cmds.SetBodyLayoutMsg:
		m.panelWidth = msg.LeftWidth
		m.panelHeight = msg.Height
		m.resizeList()

	case tea.KeyPressMsg:
		// The inner list still gets the key below - that is where the filter
		// input lives - but none of the panel's own verbs fire while it is
		// being typed into. Both body panels are always active now, so the
		// panel answers every key (the details panel handles the docker
		// verbs; this list handles its own management verbs).
		if m.OwnsKeyboard() {
			break
		}

		switch {
		// The reserved ungrouped row has no profile tag behind it, so there
		// is nothing to rename, delete, or reconcile membership against - the
		// three list-management verbs refuse it. Select (start) and Stop
		// still work on it: that is the point of the row.
		case key.Matches(msg, keys.List.Delete):
			if selectedGroup, ok := m.list.SelectedItem().(apptypes.GroupListItem); ok &&
				selectedGroup.Name != apptypes.UngroupedGroup {
				finalCmds = append(finalCmds, cmds.OpenDeleteGroupModal(selectedGroup.Name))
			}

		case key.Matches(msg, keys.List.Edit):
			if selectedGroup, ok := m.list.SelectedItem().(apptypes.GroupListItem); ok &&
				selectedGroup.Name != apptypes.UngroupedGroup {
				finalCmds = append(finalCmds, cmds.OpenEditGroupModal(selectedGroup.Name))
			}

		case key.Matches(msg, keys.List.Rename):
			if selectedGroup, ok := m.list.SelectedItem().(apptypes.GroupListItem); ok &&
				selectedGroup.Name != apptypes.UngroupedGroup {
				finalCmds = append(finalCmds, cmds.OpenRenameGroupModal(selectedGroup.Name))
			}

		// A toggles the reserved ungrouped row's materialization. The list
		// only knows the row is selected; AppModel decides adopt vs release
		// from the loaded project.
		case key.Matches(msg, keys.List.AdoptUngrouped, keys.List.ReleaseUngrouped):
			if selectedGroup, ok := m.list.SelectedItem().(apptypes.GroupListItem); ok &&
				selectedGroup.Name == apptypes.UngroupedGroup {
				finalCmds = append(finalCmds, cmds.RequestToggleUngrouped())
			}
		}

	case cmds.SetHomeStatsMsg:
		m.stats = msg
		m.hasStats = true
		// The footer appearing takes a row away from the list.
		m.resizeList()

	// AppModel decides which group is selected after a config reload, so the
	// list follows that decision rather than keeping its own.
	case cmds.SetSelectedGroupMsg:
		m.activeGroup = string(msg)
		m.syncActiveIndex()

	case cmds.SetGroupsListMsg:
		groupsList := []list.Item{}

		for _, group := range msg {
			newGroup := apptypes.GroupListItem{
				Name:    group.Name,
				Running: group.Running,
				Total:   group.Total,
			}

			groupsList = append(groupsList, newGroup)
		}

		cmd := m.list.SetItems(groupsList)
		finalCmds = append(finalCmds, cmd)
		m.syncActiveIndex()
	}

	// Both body panels are always active now, so the inner list always
	// receives the keystroke. Track cursor before the list processes the key,
	// so we can detect movement and auto-select the item under it.
	previousIndex := m.list.Index()

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	finalCmds = append(finalCmds, cmd)

	// Auto-select: if the cursor moved, select the item under it.
	if m.list.Index() != previousIndex {
		if item := m.list.SelectedItem(); item != nil {
			if group, ok := item.(apptypes.GroupListItem); ok {
				m.activeGroup = group.Name
				m.syncActiveIndex()
				finalCmds = append(finalCmds, cmds.SetSelectedGroup(group.Name))
			}
		}
	}

	if state := m.list.FilterState(); state != filterStateBefore {
		finalCmds = append(finalCmds, cmds.SetListFilterState(state))
	}

	return m, tea.Batch(finalCmds...)
}
