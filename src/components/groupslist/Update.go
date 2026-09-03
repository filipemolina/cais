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
// no row at all when it isn't on screen. Runs on both the list and the
// selection changing, since they arrive as unordered separate messages.
//
// It walks VisibleItems, not Items: the delegate is handed the index of the
// row it is drawing, and list.populatedView numbers those against the
// visible slice. Under a filter the two disagree - see the same note on
// serviceslist.Model.syncActiveIndex.
func (m *Model) syncActiveIndex() {
	active := -1

	for i, item := range m.list.VisibleItems() {
		if group, ok := item.(apptypes.GroupListItem); ok && group.Name == m.activeGroup {
			active = i
			break
		}
	}

	if active == m.listDelegate.activeIndex {
		return
	}

	m.listDelegate.activeIndex = active
	m.list.SetDelegate(m.listDelegate)
}

// cursorGroup returns the name of the group the cursor is sitting on, in the
// list's own visible space, and whether there is one at all. An empty filter
// result has no row under the cursor and reports false.
func cursorGroup(l list.Model) (string, bool) {
	item, ok := l.SelectedItem().(apptypes.GroupListItem)
	if !ok {
		return "", false
	}

	return item.Name, true
}

// setItems replaces the list's rows, noting that the cursor will need putting
// back afterwards when a filter is standing. See restoreCursor.
func (m *Model) setItems(items []list.Item) tea.Cmd {
	if m.list.FilterState() != list.Unfiltered {
		m.cursorNeedsRestore = true
	}

	return m.list.SetItems(items)
}

// restoreCursor puts the cursor back on activeGroup once the filtered rows
// return after a rebuild - see the fuller note on
// serviceslist.Model.restoreCursor. The groups list rebuilds on the same
// five-second tick, since a group's running count changes with its
// containers.
func (m *Model) restoreCursor() {
	if !m.cursorNeedsRestore || len(m.list.VisibleItems()) == 0 {
		return
	}

	for i, item := range m.list.VisibleItems() {
		if group, ok := item.(apptypes.GroupListItem); ok && group.Name == m.activeGroup {
			m.list.Select(i)
			break
		}
	}

	m.cursorNeedsRestore = false
}

// applyGroups puts a freshly derived group list on screen, in place where it
// can be and wholesale where it cannot.
//
// This one message is both the five-second refresh and the reload after the
// compose file changes. The refresh almost always brings the same groups in
// the same order with only their running counts moved, and for that case
// list.SetItem rewrites the rows without disturbing anything else. Replacing
// the lot with list.SetItems would nil filteredItems and defer the rebuild to
// a command, leaving a standing filter with no rows for a whole message cycle
// - long enough to be rendered, which made a filtered list flash "No groups."
// every five seconds. See serviceslist.Model.updateServiceStatuses for the
// same treatment on the other panel.
//
// A reload that adds, removes or reorders groups cannot be expressed as
// in-place writes, so it goes through setItems and accepts the blank cycle;
// restoreCursor is what puts the cursor back afterwards.
func (m *Model) applyGroups(groups []list.Item) tea.Cmd {
	if !m.sameGroupsInOrder(groups) {
		return m.setItems(groups)
	}

	var refilter tea.Cmd

	for i, item := range groups {
		if item == m.list.Items()[i] {
			continue
		}

		if cmd := m.list.SetItem(i, item); cmd != nil {
			refilter = cmd
		}
	}

	return refilter
}

// sameGroupsInOrder reports whether the incoming groups are the ones already
// on the list, in the same positions - which is what makes an in-place
// rewrite equivalent to a replacement. Only the names are compared: the
// counts are exactly what a refresh is expected to change.
func (m *Model) sameGroupsInOrder(groups []list.Item) bool {
	current := m.list.Items()
	if len(current) != len(groups) {
		return false
	}

	for i, item := range groups {
		incoming, ok := item.(apptypes.GroupListItem)
		if !ok {
			return false
		}

		existing, ok := current[i].(apptypes.GroupListItem)
		if !ok || existing.Name != incoming.Name {
			return false
		}
	}

	return true
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

		cmd := m.applyGroups(groupsList)
		finalCmds = append(finalCmds, cmd)
		m.syncActiveIndex()
	}

	// Both body panels are always active now, so the inner list always
	// receives the keystroke. Note which group the cursor is on before the
	// list processes the message, so the row changing underneath it can be
	// told apart from the selection being set from outside.
	previousGroup, _ := cursorGroup(m.list)

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	finalCmds = append(finalCmds, cmd)

	// bubbles draws the filter input only while it is being typed, so once a
	// filter is accepted nothing on screen says the list is still filtered.
	// Its status bar does - see serviceslist.Model.Update. Toggled here,
	// before anything reads the cursor, because turning it on costs the list
	// a row and so re-runs its pagination.
	filterState := m.list.FilterState()
	if filterState != filterStateBefore {
		m.list.SetShowStatusBar(filterState == list.FilterApplied)
	}

	// Auto-select: if a different group is under the cursor now, select it.
	// The test is identity rather than list.Index() - see the fuller note on
	// serviceslist.Model.Update for why an index comparison misses filtering.
	// Suspended across a rebuild under a filter - see the note on
	// serviceslist.Model.Update. Every unfiltered path is untouched, since
	// nothing sets the flag there.
	if !m.cursorNeedsRestore {
		if group, ok := cursorGroup(m.list); ok && group != previousGroup {
			m.activeGroup = group
			finalCmds = append(finalCmds, cmds.SetSelectedGroup(group))
		}
	}

	m.restoreCursor()

	// Which rows are visible can change without the selection changing, and
	// the delegate's index is relative to those rows.
	m.syncActiveIndex()

	if filterState != filterStateBefore {
		finalCmds = append(finalCmds, cmds.SetListFilterState(filterState))
	}

	return m, tea.Batch(finalCmds...)
}
