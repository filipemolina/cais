package components

import (
	"fmt"
	"image/color"
	"io"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

/*
 * Styling by creating a custom delegate
 */

type servicesListCustomDelegate struct {
	isParentFocused bool
	activeIndex     int
}

func (d servicesListCustomDelegate) Height() int                             { return 4 }
func (d servicesListCustomDelegate) Spacing() int                            { return 0 }
func (d servicesListCustomDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

// Render handles the actual drawing of the item
func (d servicesListCustomDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	// Cast the generic list.Item back to our specific ServiceListItem
	item, ok := listItem.(apptypes.ServiceListItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()
	isActive := index == d.activeIndex
	var titleColor color.Color

	if isActive {
		titleColor = appstyles.Active.TextPrimary
	} else {
		titleColor = appstyles.Active.TextMuted
	}

	rowBg := listRowBg(isActive, d.isParentFocused)

	wrapperStyle := lipgloss.NewStyle().
		Width(m.Width()).
		Padding(1).
		Background(rowBg)

	titleStyle := lipgloss.NewStyle().
		Bold(isActive).
		Foreground(titleColor).
		Background(rowBg).
		Width(m.Width())

	if isActive {
		wrapperStyle = wrapperStyle.
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderLeftForeground(appstyles.Active.Accent)

	} else if isSelected && d.isParentFocused {
		wrapperStyle = wrapperStyle.
			Bold(true).
			BorderLeft(true).
			BorderStyle(lipgloss.DoubleBorder()).
			BorderLeftForeground(appstyles.Active.TextPrimary)

	} else {
		// Default unselected, inactive state
		wrapperStyle = wrapperStyle.
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderLeftForeground(appstyles.Active.TextMuted)

	}

	title := titleStyle.Render(item.Title())
	description := item.Description(isActive)

	// Seal the row against its own background before handing it to the list:
	// JoinVertical pads the description out to the title's width with unstyled
	// spaces, which would otherwise show the terminal background through the
	// row. Sealing here (rather than over the whole list) is what keeps the
	// active row's lighter surface color from being flattened to the panel's.
	row := appstyles.FillBackground(rowBg, wrapperStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, description)))

	// Print the styled string to the Bubble Tea io.Writer
	fmt.Fprint(w, row)
}

/*
 * Implementation of tea.Model
 */

type ServicesListModel struct {
	list         list.Model
	listDelegate servicesListCustomDelegate
	// activeService is the name of the service the cursor is on. The cursor
	// position IS the selection now (auto-select on navigation), so a stored
	// row number would highlight whichever service moved into that row.
	activeService string
	isFocused     bool
	componentId   int
	fileName      string
	project       *types.Project
	panelWidth    int
	panelHeight   int
	// containers is the latest known container list, used to derive the
	// RUNNING/STOPPED status shown on each service row.
	containers []apptypes.DockerContainer
}

// syncActiveIndex points the delegate at the row holding activeService, or
// at no row at all when it isn't in the list.
//
// It runs on both the list and the selection changing, because the two
// arrive as separate messages and tea.Batch makes no promise about their
// order. Re-deriving from the name on each means the pair converges on the
// right row whichever lands first.
func (m *ServicesListModel) syncActiveIndex() {
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

func (m ServicesListModel) Init() tea.Cmd {
	return nil
}

// OwnsKeyboard reports whether the list is taking every keystroke for itself,
// which it does while a filter is being typed. Same rule as the groups list -
// see GroupListModel.OwnsKeyboard.
func (m ServicesListModel) OwnsKeyboard() bool {
	return m.list.FilterState() == list.Filtering
}

// KeepsEsc reports whether the list needs esc for itself. Same rule as the
// groups list - see GroupListModel.KeepsEsc.
func (m ServicesListModel) KeepsEsc() bool {
	return m.isFocused && m.list.FilterState() == list.FilterApplied
}

// FilterState exposes how much of the keyboard the list has taken. Same rule
// as the groups list - see GroupListModel.FilterState.
func (m ServicesListModel) FilterState() list.FilterState {
	return m.list.FilterState()
}

// resizeList sizes the inner list to the space left inside the panel box
// after the wrapper padding.
func (m *ServicesListModel) resizeList() {
	h, v := listWrapperStyle.GetFrameSize()

	m.list.SetSize(
		max(0, m.panelWidth-h),
		max(0, m.panelHeight-v),
	)
}

func (m ServicesListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var finalCmds []tea.Cmd

	// See GroupListModel.Update: the footer's keys depend on this, so a change
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
		if msg.Err == nil {
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

func (m ServicesListModel) View() tea.View {
	// Same 3-tier treatment as the groups list: focus lifts the panel from
	// tier 3 to tier 4 rather than adding a border, so the panel's box stays
	// the same size whether or not it is focused.
	bg := panelBg(m.isFocused)

	wrapper := fitBox(listWrapperStyle.Background(bg), m.panelWidth, m.panelHeight)

	// The list joins its title, rows and paginator internally, padding the
	// short ones with unstyled spaces; seal them against the panel tier. Rows
	// arrive already sealed against their own background, so this only fills
	// what the list itself left bare.
	v := tea.NewView(wrapper.Render(appstyles.FillBackground(bg, m.list.View())))
	return v
}

/*
 * Initializer function
 */

// buildItems converts a slice of service configs into list items, picking up
// the latest container state from the model so each row shows the correct
// RUNNING/STOPPED pill and memory usage.
func (m *ServicesListModel) buildItems(services []types.ServiceConfig) []list.Item {
	items := make([]list.Item, 0, len(services))

	for _, service := range services {
		item := apptypes.ServiceListItem{
			Service:  service,
			Status:   m.containerStatus(service.Name),
			MemUsage: m.containerMemUsage(service.Name),
		}

		items = append(items, item)
	}

	return items
}

// containerStatus returns "running", "stopped", or "" depending on whether a
// live container exists for the given compose service name.
func (m *ServicesListModel) containerStatus(serviceName string) string {
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

// containerMemUsage returns the memory usage string for the given service,
// formatted as "Usage (Percent)" (e.g., "21.71MiB (0.07%)"), or "" if no
// container exists or stats are unavailable.
func (m *ServicesListModel) containerMemUsage(serviceName string) string {
	for _, c := range m.containers {
		if c.Service == serviceName && c.State == "running" {
			return formatMemUsage(c.MemUsage, c.MemPerc)
		}
	}
	return ""
}

// formatMemUsage formats memory usage as "Usage (Percent)", e.g.,
// "21.71MiB / 31.02GiB" + "0.07%" -> "21.71MiB (0.07%)".
// If percent is empty, returns just the usage part (before "/").
func formatMemUsage(memUsage, memPerc string) string {
	usage := memUsage
	if idx := strings.Index(memUsage, "/"); idx != -1 {
		usage = strings.TrimSpace(memUsage[:idx])
	}
	if memPerc != "" {
		return usage + " (" + memPerc + ")"
	}
	return usage
}

// updateServiceStatuses refreshes the status and memory fields on every
// list item to match the current container state. Called whenever a
// GetRunningContainersMsg or GetContainerStatsMsg arrives with fresh data.
// It returns a tea.Cmd so that any filter re-application triggered by
// SetItems (required when a filter is active) gets executed by the
// runtime, keeping the filtered view consistent.
func (m *ServicesListModel) updateServiceStatuses() tea.Cmd {
	items := m.list.Items()
	updated := make([]list.Item, 0, len(items))

	for _, item := range items {
		svcItem, ok := item.(apptypes.ServiceListItem)
		if !ok {
			updated = append(updated, item)
			continue
		}

		svcItem.Status = m.containerStatus(svcItem.Service.Name)
		svcItem.MemUsage = m.containerMemUsage(svcItem.Service.Name)
		updated = append(updated, svcItem)
	}

	return m.list.SetItems(updated)
}

func ServicesList(services []types.ServiceConfig, width int, height int) tea.Model {
	model := ServicesListModel{
		componentId: 1,
	}

	items := model.buildItems(services)

	// -1 rather than the zero value: no service is active until one is
	// selected, and 0 would render the first row as though one were.
	listDelegate := servicesListCustomDelegate{activeIndex: -1}
	servicesList := list.New(items, listDelegate, width, height)
	servicesList.SetShowHelp(false)
	servicesList.SetShowStatusBar(false)
	// See keys.ListKeyMap: the default map's letter aliases for paging collide
	// with the panel verbs.
	servicesList.KeyMap = keys.ListKeyMap()

	servicesList.Title = "Services"
	servicesList.Paginator.ActiveDot = " ● "
	servicesList.Paginator.InactiveDot = " ○ "
	servicesList.Styles.Title = servicesList.
		Styles.
		Title.
		Background(appstyles.Active.Accent)

	model.list = servicesList
	model.listDelegate = listDelegate

	return model
}
