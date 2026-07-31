package serviceslist

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

// syncActiveIndex points the delegate at the row holding activeService, or
// at no row at all when it isn't in the list.
//
// It runs on both the list and the selection changing, because the two
// arrive as separate messages and tea.Batch makes no promise about their
// order. Re-deriving from the name on each means the pair converges on the
// right row whichever lands first.
func (m *Model) syncActiveIndex() {
	active := -1

	for i, item := range m.list.Items() {
		if service, ok := item.(apptypes.ServiceListItem); ok && service.Service.Name == m.activeService {
			active = i
			break
		}
	}

	m.listDelegate.activeIndex = active
	m.list.SetDelegate(m.listDelegate)
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
		// Space/Enter starts the selected service (quick action).
		// Selection happens automatically on cursor movement.
		if m.isFocused && !m.OwnsKeyboard() && key.Matches(msg, keys.List.Select) {
			if m.activeService != "" {
				finalCmds = append(finalCmds, cmds.RequestDockerAction("start", m.activeService, false))
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

	case cmds.SetFocusMsg:
		if int(msg) == m.componentId {
			m.isFocused = true
			m.listDelegate.isParentFocused = true
			m.list.SetDelegate(m.listDelegate)
		} else {
			m.isFocused = false
			m.listDelegate.isParentFocused = false
			m.list.SetDelegate(m.listDelegate)
		}
	}

	if m.isFocused {
		// Track cursor before the list processes the key, so we can detect
		// movement and auto-select the item under it.
		previousIndex := m.list.Index()

		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		finalCmds = append(finalCmds, cmd)

		// Auto-select: if the cursor moved, select the item under it.
		if m.list.Index() != previousIndex {
			if item := m.list.SelectedItem(); item != nil {
				if service, ok := item.(apptypes.ServiceListItem); ok {
					m.activeService = service.Service.Name
					m.syncActiveIndex()
					finalCmds = append(finalCmds, cmds.SetSelectedService(service.Service))
				}
			}
		}
	}

	if state := m.list.FilterState(); state != filterStateBefore {
		finalCmds = append(finalCmds, cmds.SetListFilterState(state))
	}

	return m, tea.Batch(finalCmds...)
}
