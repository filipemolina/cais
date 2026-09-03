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

// containerStatus returns "running", "stopped", or "" depending on whether a
// live container exists for the given compose service name.
func (m *Model) containerStatus(serviceName string) string {
	for _, c := range m.containers {
		if c.Service == serviceName {
			if c.State == "running" {
				return "running"
			}
			return "stopped"
		}
	}
	return ""
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

// updateServiceStatuses refreshes the status and memory fields on every
// list item to match the current container state. Called whenever a
// GetRunningContainersMsg or GetContainerStatsMsg arrives with fresh data.
// It returns a tea.Cmd so that any filter re-application triggered by
// SetItems (required when a filter is active) gets executed by the
// runtime, keeping the filtered view consistent.
func (m *Model) updateServiceStatuses() tea.Cmd {
	items := m.list.Items()
	updated := make([]list.Item, 0, len(items))

	for _, item := range items {
		svcItem, ok := item.(apptypes.ServiceListItem)
		if !ok {
			updated = append(updated, item)
			continue
		}

		svcItem.Status = m.containerStatus(svcItem.Service.Name)
		svcItem.MemUsage, svcItem.MemPerc = m.containerMem(svcItem.Service.Name)
		updated = append(updated, svcItem)
	}

	return m.list.SetItems(updated)
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

		cmd := m.list.SetItems(servicesList)
		finalCmds = append(finalCmds, cmd)
		m.syncActiveIndex()

	case cmds.GetRunningContainersMsg:
		if msg.Err == nil {
			m.containers = msg.Containers
			finalCmds = append(finalCmds, m.updateServiceStatuses())
		}

	case cmds.GetContainerStatsMsg:
		// Present-but-unenriched still beats stale: a failed stats call sends
		// the containers through without their runtime numbers, so the status
		// column stays correct even when the memory column empties.
		if msg.Containers != nil {
			m.containers = msg.Containers
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
	if service, ok := cursorService(m.list); ok && service.Name != previousService.Name {
		m.activeService = service.Name
		finalCmds = append(finalCmds, cmds.SetSelectedService(service))
	}

	// Which rows are visible can change without the selection changing - a
	// filter keystroke, or a stats poll re-applying the filter - and the
	// delegate's index is relative to those rows. Re-point it on every
	// message rather than only on the two that used to be enough.
	m.syncActiveIndex()

	if state := m.list.FilterState(); state != filterStateBefore {
		finalCmds = append(finalCmds, cmds.SetListFilterState(state))
	}

	return m, tea.Batch(finalCmds...)
}
