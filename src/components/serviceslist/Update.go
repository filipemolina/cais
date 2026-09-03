package serviceslist

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/keys"
)

// syncActiveIndex points the delegate at the row holding activeService, or
// at no row at all when it isn't on screen.
//
// It runs on both the list and the selection changing, because the two
// arrive as separate messages and tea.Batch makes no promise about their
// order. Re-deriving from the name on each means the pair converges on the
// right row whichever lands first.
//
// It walks VisibleItems, not Items: the delegate is handed the index of the
// row it is drawing, and list.populatedView numbers those against the
// visible slice. Under a filter the two disagree, and scanning the whole
// slice put the highlight on the wrong row - or, for a name that filtered
// out to a position past the end of the visible list, on no row at all.
func (m *Model) syncActiveIndex() {
	active := -1

	for i, item := range m.list.VisibleItems() {
		if service, ok := item.(apptypes.ServiceListItem); ok && service.Service.Name == m.activeService {
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

// cursorService returns the service the cursor is sitting on, in the list's
// own visible space, and whether there is one at all. An empty filter result
// has no row under the cursor and reports false.
func cursorService(l list.Model) (types.ServiceConfig, bool) {
	item, ok := l.SelectedItem().(apptypes.ServiceListItem)
	if !ok {
		return types.ServiceConfig{}, false
	}

	return item.Service, true
}

// setItems replaces the list's rows, noting that the cursor will need putting
// back afterwards when a filter is standing. See restoreCursor.
func (m *Model) setItems(items []list.Item) tea.Cmd {
	if m.list.FilterState() != list.Unfiltered {
		m.cursorNeedsRestore = true
	}

	return m.list.SetItems(items)
}

// restoreCursor puts the cursor back on activeService once the filtered rows
// return after a rebuild.
//
// list.SetItems blanks filteredItems and only re-applies the filter a message
// later, so for one cycle a filtered list has no visible rows at all. The
// list clamps its cursor into that empty range as it goes past
// (list.handleBrowsing's closing clamp against maxCursorIndex), which walked
// the selection back to the first match every time the five-second container
// refresh landed on a filtered list. The rows come back intact; only the
// cursor is lost, so it is put back by name.
func (m *Model) restoreCursor() {
	if !m.cursorNeedsRestore || len(m.list.VisibleItems()) == 0 {
		return
	}

	for i, item := range m.list.VisibleItems() {
		if service, ok := item.(apptypes.ServiceListItem); ok && service.Service.Name == m.activeService {
			m.list.Select(i)
			break
		}
	}

	m.cursorNeedsRestore = false
}

// resizeList sizes the inner list to the space left inside the panel box
// after the wrapper padding.
func (m *Model) resizeList() {
	h, v := chrome.ListWrapperStyle.GetFrameSize()

	m.list.SetSize(
		max(0, m.panelWidth-h),
		max(0, m.panelHeight-v),
	)
}

// buildItems converts a slice of service configs into list items, picking up
// the latest container state from the model so each row shows the correct
// RUNNING/STOPPED pill and memory usage.
func (m *Model) buildItems(services []types.ServiceConfig) []list.Item {
	items := make([]list.Item, 0, len(services))

	for _, service := range services {
		usage, perc := m.containerMem(service.Name)
		item := apptypes.ServiceListItem{
			Service:  service,
			Status:   m.containerStatus(service.Name),
			MemUsage: usage,
			MemPerc:  perc,
		}

		items = append(items, item)
	}

	return items
}

// containerStatus returns "running", "stopped", or "" for a compose service,
// where "" means unknown rather than absent.
//
// A service with no container of its own is stopped, but only once docker has
// been asked - before that every service looks container-less and reporting
// them all stopped is a guess dressed up as a fact. containersKnown is what
// tells the two apart.
func (m *Model) containerStatus(serviceName string) string {
	if !m.containersKnown {
		return ""
	}

	for _, c := range m.containers {
		if c.Service == serviceName {
			if c.State == "running" {
				return "running"
			}
			return "stopped"
		}
	}

	return "stopped"
}

// containerMem returns docker's raw memory usage and percentage for the
// given service, or two empty strings if no container exists or stats are
// unavailable. The row formats them at render time via
// apptypes.FormatMemUsage - see the note there on why they are not formatted
// here.
func (m *Model) containerMem(serviceName string) (usage, perc string) {
	for _, c := range m.containers {
		if c.Service == serviceName && c.State == "running" {
			return c.MemUsage, c.MemPerc
		}
	}
	return "", ""
}

// updateServiceStatuses refreshes the status and memory fields on every list
// item to match the current container state. Called whenever a
// GetRunningContainersMsg or GetContainerStatsMsg arrives with fresh data.
//
// It writes each row through list.SetItem rather than replacing the lot with
// list.SetItems, which matters only when a filter is standing but matters a
// lot there. SetItems nils filteredItems and hands back a command to rebuild
// them, so the rows are gone for a whole message cycle - long enough to be
// rendered, which made a filtered list flash "No services." every five
// seconds. SetItem leaves filteredItems alone: the rows stay on screen
// carrying last tick's numbers until the re-filter lands and swaps in this
// tick's. Stale for one frame beats absent for one frame.
//
// Only the last command is kept. Each SetItem returns its own re-filter
// command, and every one of them would recompute the same thing; the last is
// the only one that sees every row already written, so it is the only one
// worth running.
//
// This rewrites rows in place - same rows, same order, only their status and
// memory fields change - which is what makes SetItem applicable at all. A
// path that adds or removes services cannot use it and must go through
// setItems; see the SetServicesListMsg case.
func (m *Model) updateServiceStatuses() tea.Cmd {
	var refilter tea.Cmd

	for i, item := range m.list.Items() {
		svcItem, ok := item.(apptypes.ServiceListItem)
		if !ok {
			continue
		}

		status := m.containerStatus(svcItem.Service.Name)
		usage, perc := m.containerMem(svcItem.Service.Name)

		if svcItem.Status == status && svcItem.MemUsage == usage && svcItem.MemPerc == perc {
			continue
		}

		svcItem.Status = status
		svcItem.MemUsage, svcItem.MemPerc = usage, perc

		if cmd := m.list.SetItem(i, svcItem); cmd != nil {
			refilter = cmd
		}
	}

	return refilter
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var finalCmds []tea.Cmd

	// See groupslist.Model.Update: the footer's keys depend on this, so a change
	// has to be broadcast.
	filterStateBefore := m.list.FilterState()

	switch msg := msg.(type) {
	// Sizing comes from AppModel, not WindowSizeMsg: the Services page is never
	// the active page when the terminal is first measured, so a resize-derived
	// height left this list a few rows tall showing a single service.
	case cmds.SetBodyLayoutMsg:
		m.panelWidth = msg.LeftWidth
		m.panelHeight = msg.Height
		m.resizeList()

	case tea.KeyPressMsg:
		// The inner list still gets the key below - that is where the filter
		// input lives - but neither quick action fires while it is being
		// typed into. Both body panels are always active now, so the panel
		// answers every key (the details panel handles the docker verbs).
		if m.OwnsKeyboard() {
			break
		}

		switch {
		case key.Matches(msg, keys.List.Delete):
			// d deletes the service's whole entry from the compose file -
			// same key, same "delete the highlighted thing" meaning as the
			// groups list's d, widened rather than given a second key. Goes
			// through a confirm modal because, unlike a group delete, this
			// removes a service definition rather than just a tag.
			if m.activeService != "" {
				finalCmds = append(finalCmds, cmds.OpenDeleteServiceModal(m.activeService))
			}
		}

	// AppModel decides which service is selected after a config reload, so
	// the list follows that decision rather than keeping its own.
	case cmds.SetSelectedServiceMsg:
		m.activeService = types.ServiceConfig(msg).Name
		m.syncActiveIndex()

	case cmds.SetServicesListMsg:
		servicesList := m.buildItems(msg)

		cmd := m.setItems(servicesList)
		finalCmds = append(finalCmds, cmd)
		m.syncActiveIndex()

	case cmds.GetRunningContainersMsg:
		if msg.Err == nil {
			m.containers = msg.Containers
			m.containersKnown = true
			finalCmds = append(finalCmds, m.updateServiceStatuses())
		}

	case cmds.GetContainerStatsMsg:
		// Present-but-unenriched still beats stale: a failed stats call sends
		// the containers through without their runtime numbers, so the status
		// column stays correct even when the memory column empties.
		if msg.Containers != nil {
			m.containers = msg.Containers
			m.containersKnown = true
			finalCmds = append(finalCmds, m.updateServiceStatuses())
		}

	}

	// Both body panels are always active now, so the inner list always
	// receives the keystroke. Note which service the cursor is on before the
	// list processes the message, so the row changing underneath it can be
	// told apart from the selection being set from outside.
	previousService, _ := cursorService(m.list)

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	finalCmds = append(finalCmds, cmd)

	// bubbles draws the filter input only while it is being typed, so once a
	// filter is accepted nothing on screen says the list is still filtered.
	// Its status bar does - the term, the match count and how many rows are
	// hidden - so it is shown for exactly as long as a filter is standing.
	// Toggled here, before anything reads the cursor, because turning it on
	// costs the list a row and so re-runs its pagination.
	filterState := m.list.FilterState()
	if filterState != filterStateBefore {
		m.list.SetShowStatusBar(filterState == list.FilterApplied)
	}

	// Auto-select: if a different service is under the cursor now, select it.
	//
	// The test is the service's identity, not list.Index(). Filtering moves
	// rows out from under a cursor that never moves - applying a filter sends
	// the cursor to the top, so a cursor already at the top keeps the same
	// index while row 0 becomes an entirely different service. Comparing
	// indexes missed that and left the details panel showing a service the
	// filter had just hidden. Identity also covers the list being reordered
	// or reloaded under a stationary cursor.
	//
	// A selection arriving from AppModel (SetSelectedServiceMsg, handled
	// above) does not move the cursor, so previousService already matches and
	// nothing is echoed back.
	// Suspended across a rebuild under a filter: the rows are blank for a
	// cycle and then come back, which is not the user moving anywhere. Left
	// running, it read the blank-then-back as navigation and published the
	// first match on every refresh. Every unfiltered path is untouched, since
	// nothing sets the flag there.
	if !m.cursorNeedsRestore {
		if service, ok := cursorService(m.list); ok && service.Name != previousService.Name {
			m.activeService = service.Name
			finalCmds = append(finalCmds, cmds.SetSelectedService(service))
		}
	}

	m.restoreCursor()

	// Which rows are visible can change without the selection changing - a
	// filter keystroke, or a stats poll re-applying the filter - and the
	// delegate's index is relative to those rows. Re-point it on every
	// message rather than only on the two that used to be enough.
	m.syncActiveIndex()

	if filterState != filterStateBefore {
		finalCmds = append(finalCmds, cmds.SetListFilterState(filterState))
	}

	return m, tea.Batch(finalCmds...)
}
