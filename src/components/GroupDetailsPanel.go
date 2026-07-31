package components

import (
	"fmt"
	"image/color"
	"slices"
	"strings"

	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/cmds"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/stack-stitcher/src/constants"
	"github.com/filipemolina/stack-stitcher/src/keys"
	"github.com/mattn/go-runewidth"
)

type GroupDetailsPanelModel struct {
	selectedGroup string
	services      []types.ServiceConfig
	containers    []apptypes.DockerContainer
	panelWidth    int
	panelHeight   int
	isFocused     bool
	componentId   int
	pendingAction *PendingAction
	spinner       spinner.Model
}

func (m GroupDetailsPanelModel) Init() tea.Cmd {
	return nil
}

func (m GroupDetailsPanelModel) memberServices() []types.ServiceConfig {
	var members []types.ServiceConfig

	for _, service := range m.services {
		if slices.Contains(service.Profiles, m.selectedGroup) {
			members = append(members, service)
		}
	}

	return members
}

// knownGroups returns every distinct Compose profile referenced by the
// loaded services. It distinguishes the "no groups exist yet" onboarding
// state from "groups exist but nothing is selected".
func (m GroupDetailsPanelModel) knownGroups() []string {
	seen := make(map[string]bool)
	var groups []string

	for _, service := range m.services {
		for _, profile := range service.Profiles {
			if !seen[profile] {
				seen[profile] = true
				groups = append(groups, profile)
			}
		}
	}

	return groups
}

func (m GroupDetailsPanelModel) isServiceRunning(serviceName string) bool {
	for _, container := range m.containers {
		if container.Service == serviceName {
			return container.State == "running"
		}
	}

	return false
}

// containerForService finds the live container whose Service label matches
// the given compose service name. Returns false when the service has no
// created/running container yet, so the row renders as stopped.
func (m GroupDetailsPanelModel) containerForService(serviceName string) (apptypes.DockerContainer, bool) {
	for _, container := range m.containers {
		if container.Service == serviceName {
			return container, true
		}
	}

	return apptypes.DockerContainer{}, false
}

func (m GroupDetailsPanelModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var finalCmds []tea.Cmd

	switch msg := msg.(type) {
	// AppModel emits this on resize, page switch, and error-banner changes,
	// so the panel always fills the exact body region. Deriving the width
	// from WindowSizeMsg instead would leave it at 0 until Home happened to
	// be the active page during a resize.
	case cmds.SetBodyLayoutMsg:
		m.panelWidth = msg.RightWidth
		m.panelHeight = msg.Height

	case cmds.SetPendingActionMsg:
		m.pendingAction = &PendingAction{Action: msg.Action, Target: msg.Target, IsGroup: msg.IsGroup}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(m.spinner.Tick())
		finalCmds = append(finalCmds, cmd)

	case cmds.ClearPendingActionMsg:
		m.pendingAction = nil

	case spinner.TickMsg:
		if m.pendingAction != nil {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			finalCmds = append(finalCmds, cmd)
		}

	case cmds.SetFocusMsg:
		if int(msg) == m.componentId {
			m.isFocused = true
		} else {
			m.isFocused = false
		}

	case cmds.SetSelectedGroupMsg:
		m.selectedGroup = string(msg)

	case cmds.SetServicesListMsg:
		m.services = msg

	case cmds.GetRunningContainersMsg:
		if msg.Err == nil {
			m.containers = msg.Containers
		}

	// A background poll withholds GetRunningContainersMsg while stats are in
	// flight and delivers the containers here instead, so this panel has to
	// answer both or its member rows only refresh on a foreground reload.
	case cmds.GetContainerStatsMsg:
		if msg.Containers != nil {
			m.containers = msg.Containers
		}

	case tea.KeyPressMsg:
		if !m.isFocused || m.selectedGroup == "" {
			break
		}

		if action, ok := dockerActionFor(msg); ok {
			actionCmd := cmds.RequestDockerAction(action, m.selectedGroup, true)
			finalCmds = append(finalCmds, actionCmd)
		}

		switch {
		case key.Matches(msg, keys.Details.Remove):
			// Remove destroys containers, so it goes through a
			// confirmation first, unlike the other four actions.
			finalCmds = append(finalCmds, cmds.OpenConfirmModal(
				fmt.Sprintf("Remove group %q?\nThis stops and removes its containers.", m.selectedGroup),
				cmds.RequestDockerAction("remove", m.selectedGroup, true),
			))

		case key.Matches(msg, keys.Details.Logs):
			finalCmds = append(finalCmds, cmds.OpenLogsModal(m.selectedGroup, true))
		}
	}

	return m, tea.Batch(finalCmds...)
}

// View renders the panel body and hands it to renderPanelFrame. The body
// is always non-empty (empty states draw their own cards). The action
// buttons are the last block of the body, which pins them to the bottom of
// the panel.
func (m GroupDetailsPanelModel) View() tea.View {
	body := m.renderBody()
	screen := renderPanelFrame("Details", m.titlePill(), m.isFocused, m.panelWidth, m.panelHeight, body)

	return tea.NewView(screen)
}

// titlePill is the selected group's status pill, which rides on the panel's
// title row. Empty while the panel is showing an empty state: there is no
// group whose status it could report.
func (m GroupDetailsPanelModel) titlePill() string {
	if m.selectedGroup == "" || len(m.knownGroups()) == 0 {
		return ""
	}

	members := m.memberServices()

	return statusPill(m.runningCount(members), len(members))
}

func (m GroupDetailsPanelModel) runningCount(members []types.ServiceConfig) int {
	running := 0

	for _, svc := range members {
		if m.isServiceRunning(svc.Name) {
			running++
		}
	}

	return running
}

// actionContext is this panel's screen state in the shape keys.Active reads -
// the Home page twin of DetailsPanelModel.actionContext, and the same reasoning
// about reporting the list as focused when this pane is not.
func (m GroupDetailsPanelModel) actionContext() keys.Context {
	focused := constants.COMPONENT_BODY_LIST
	if m.isFocused {
		focused = constants.COMPONENT_BODY_DETAILS
	}

	return keys.Context{
		Page:          "Home",
		Focused:       focused,
		Selected:      m.selectedGroup != "",
		PendingAction: m.pendingAction != nil,
	}
}

// renderBody builds the panel body for the current state: onboarding,
// nothing-selected, or a selected group's header + member table + actions.
func (m GroupDetailsPanelModel) renderBody() string {
	bodyWidth := max(1, panelBodyWidth(m.panelWidth))
	bodyAvail := max(1, panelBodyHeight(m.panelHeight))
	bg := panelBg(m.isFocused)

	// No groups exist anywhere yet -> onboarding.
	if len(m.knownGroups()) == 0 {
		return renderEmptyCard(bodyWidth, bodyAvail, bg, "Getting started",
			"Groups are Compose profiles: sets of services you run together. Add a `profiles:` key to a service in your compose file to make one.",
			"n", "new group")
	}

	// Groups exist but none is selected.
	if m.selectedGroup == "" {
		return renderEmptyCard(bodyWidth, bodyAvail, bg, "Select a group",
			"Pick a group from the list to see its services.",
			"↑/↓", "to browse")
	}

	// A group is selected: header card + member table + actions.
	members := m.memberServices()
	running := m.runningCount(members)
	stopped := len(members) - running

	headerCard := m.groupHeaderCard(m.selectedGroup, running, stopped, len(members), bodyWidth)
	buttons := renderActionButtons(bodyWidth, bg, m.actionContext())

	footnoteBlock := ""
	if running == 0 && len(members) > 0 {
		footnoteBlock = lipgloss.JoinVertical(lipgloss.Left, "",
			lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).Render("Press s to start."))
	}
	footnoteH := 0
	if footnoteBlock != "" {
		footnoteH = lipgloss.Height(footnoteBlock)
	}

	hCard := lipgloss.Height(headerCard)
	buttonsH := lipgloss.Height(buttons)

	tableAvail := bodyAvail - hCard - footnoteH - buttonsH
	if tableAvail < 0 {
		tableAvail = 0
	}
	tableBlock := m.renderMemberTable(members, bodyWidth, tableAvail)

	parts := []string{headerCard, tableBlock}
	if footnoteBlock != "" {
		parts = append(parts, footnoteBlock)
	}

	// Show a spinner in the buttons area while an action is pending.
	if m.pendingAction != nil {
		parts = append(parts, m.renderPendingAction(bodyWidth, bg))
	} else {
		parts = append(parts, buttons)
	}

	bodyContent := lipgloss.JoinVertical(lipgloss.Left, parts...)

	// Safety cap: a miscount must never grow the panel past its body region.
	if bodyAvail > 0 {
		bodyContent = lipgloss.NewStyle().MaxHeight(bodyAvail).Render(bodyContent)
	}

	return bodyContent
}

// groupHeaderCard renders the selected group's name and a
// running/stopped/total summary, separated from the table by a thin rule.
// The status pill sits on the panel's title row - see titlePill.
func (m GroupDetailsPanelModel) groupHeaderCard(name string, running, stopped, total int, width int) string {
	nameRow := lipgloss.NewStyle().
		Bold(true).
		Foreground(appstyles.Active.TextPrimary).
		Width(width).
		Render(truncate(name, width))

	summary := fmt.Sprintf("%d running · %d stopped · %d services", running, stopped, total)
	summaryRow := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextMuted).
		Width(width).
		Render(summary)

	rule := lipgloss.NewStyle().
		Foreground(appstyles.Active.BorderDefault).
		Width(width).
		Render(strings.Repeat("─", max(width, 0)))

	return lipgloss.JoinVertical(lipgloss.Left, nameRow, summaryRow, rule)
}

// statusPill renders a filled pill whose color reflects the group's state:
// green when every service is running, amber when mixed, red when none run.
//
// The pill's ink (fg) does not follow the app's theme: InkOnLight/InkOnDark
// are fixed regardless of Dark, because the pill's own fill is a status color
// rather than a surface tier. What the fill *is* varies per theme, though, so
// appstyles.InkOn picks whichever of the two inks reads on it rather than
// this call site guessing - see appstyles/Contrast_test.go, which holds every
// theme's pill ink to 4.2:1.
func statusPill(running, total int) string {
	var label string
	var bg color.Color

	switch {
	case total > 0 && running == total:
		label, bg = "ALL RUNNING", appstyles.Active.StatusRunning
	case running == 0:
		label, bg = "STOPPED", appstyles.Active.StatusError
	default:
		label, bg = "MIXED", appstyles.Active.StatusStarting
	}
	fg := appstyles.InkOn(bg)

	return lipgloss.NewStyle().
		Background(bg).
		Foreground(fg).
		Bold(true).
		Padding(0, 1).
		Render(label)
}

// renderMemberTable renders the column headers, a separator, and one row
// per member service, then fills (or clips) to exactly `avail` rows so the
// action buttons stay pinned at the bottom of the panel.
func (m GroupDetailsPanelModel) renderMemberTable(members []types.ServiceConfig, width, avail int) string {
	cols := computeCols(width)

	header := renderTableHeader(cols, width)
	rule := lipgloss.NewStyle().
		Foreground(appstyles.Active.BorderDefault).
		Width(width).
		Render(strings.Repeat("─", max(width, 0)))

	parts := []string{header, rule}

	if len(members) == 0 {
		parts = append(parts, lipgloss.NewStyle().
			Foreground(appstyles.Active.TextDim).
			Width(width).
			AlignHorizontal(lipgloss.Center).
			Render("No services in this group"))
	} else {
		for _, svc := range members {
			parts = append(parts, m.renderMemberRow(cols, width, svc))
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)

	if avail < 1 {
		avail = 1
	}

	return lipgloss.NewStyle().
		Height(avail).
		MaxHeight(avail).
		Width(width).
		Render(content)
}

func (m GroupDetailsPanelModel) renderMemberRow(cols tableCols, width int, svc types.ServiceConfig) string {
	container, has := m.containerForService(svc.Name)

	state := "stopped"
	image := svc.Image
	health := "-"
	uptime := "-"
	ports := "-"
	dotColor := appstyles.Active.StatusStopped

	if has {
		state = container.State
		if container.Image != "" {
			image = container.Image
		}
		if container.HealthStatus != "" {
			health = container.HealthStatus
		}
		if container.RunningFor != "" {
			uptime = container.RunningFor
		}
		if container.Ports != "" {
			ports = container.Ports
		}
	}

	if state == "running" {
		dotColor = appstyles.Active.StatusRunning
	}

	dot := lipgloss.NewStyle().Foreground(dotColor).Width(cols.dot).Render("●")
	name := lipgloss.NewStyle().Foreground(appstyles.Active.TextPrimary).Width(cols.name).Render(truncate(svc.Name, cols.name))
	img := lipgloss.NewStyle().Foreground(appstyles.Active.TextMuted).Width(cols.image).Render(truncate(image, cols.image))
	st := lipgloss.NewStyle().Foreground(stateColor(state)).Width(cols.state).Render(truncate(state, cols.state))
	hl := lipgloss.NewStyle().Foreground(healthColor(health)).Width(cols.health).Render(truncate(health, cols.health))
	up := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).Width(cols.uptime).Render(truncate(uptime, cols.uptime))
	pt := lipgloss.NewStyle().Foreground(appstyles.Active.TextMuted).Width(cols.ports).Render(truncate(ports, cols.ports))

	row := lipgloss.JoinHorizontal(lipgloss.Left, dot, name, img, st, hl, up, pt)

	return lipgloss.NewStyle().Width(width).Render(row)
}

func renderTableHeader(cols tableCols, width int) string {
	dim := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).Bold(true)

	cells := []string{
		dim.Width(cols.dot).Render(""),
		dim.Width(cols.name).Render("NAME"),
		dim.Width(cols.image).Render("IMAGE"),
		dim.Width(cols.state).Render("STATE"),
		dim.Width(cols.health).Render("HEALTH"),
		dim.Width(cols.uptime).Render("UPTIME"),
		dim.Width(cols.ports).Render("PORTS"),
	}

	row := lipgloss.JoinHorizontal(lipgloss.Left, cells...)

	return lipgloss.NewStyle().Width(width).Render(row)
}

func stateColor(state string) color.Color {
	if state == "running" {
		return appstyles.Active.StatusRunning
	}

	return appstyles.Active.StatusStopped
}

func healthColor(health string) color.Color {
	switch health {
	case "healthy":
		return appstyles.Active.StatusRunning
	case "unhealthy":
		return appstyles.Active.StatusError
	case "starting":
		return appstyles.Active.StatusStarting
	default:
		return appstyles.Active.TextDim
	}
}

// tableCols holds the per-column widths for the member table.
type tableCols struct {
	dot, name, image, state, health, uptime, ports int
}

func (c tableCols) total() int {
	return c.dot + c.name + c.image + c.state + c.health + c.uptime + c.ports
}

// computeCols distributes the available width across the seven columns.
// Wide terminals expand the flexible columns (name, image, ports); narrow
// terminals shrink the widest column until the row fits on a single line.
func computeCols(width int) tableCols {
	if width < 1 {
		width = 1
	}

	c := tableCols{
		dot: 2, name: 18, image: 16, state: 9, health: 8, uptime: 11, ports: 16,
	}

	// Shrink the widest column until the row fits.
	for c.total() > width {
		before := c.total()
		switch widestColumn(c) {
		case "ports":
			c.ports--
		case "image":
			c.image--
		case "name":
			c.name--
		case "uptime":
			c.uptime--
		case "health":
			c.health--
		case "state":
			c.state--
		case "dot":
			c.dot--
		}
		if c.total() == before {
			break
		}
	}

	// Expand the flexible columns to fill wide terminals.
	if extra := width - c.total(); extra > 0 {
		addPorts := extra * 40 / 100
		addName := extra * 30 / 100
		addImage := extra - addPorts - addName
		c.ports += addPorts
		c.name += addName
		c.image += addImage
	}

	return c
}

func widestColumn(c tableCols) string {
	widest := "dot"
	max := c.dot

	if c.name > max {
		widest, max = "name", c.name
	}
	if c.image > max {
		widest, max = "image", c.image
	}
	if c.state > max {
		widest, max = "state", c.state
	}
	if c.health > max {
		widest, max = "health", c.health
	}
	if c.uptime > max {
		widest, max = "uptime", c.uptime
	}
	if c.ports > max {
		widest, max = "ports", c.ports
	}

	return widest
}

// truncate hard-truncates s to w display columns, appending an ellipsis
// when it is shortened. lipgloss Width wraps rather than truncates, so
// cells are pre-truncated to keep every row on a single line.
func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}

	return runewidth.Truncate(s, w, "…")
}

// renderPendingAction renders a spinner with the action description in place
// of the action buttons while a docker action is in progress.
func (m GroupDetailsPanelModel) renderPendingAction(width int, bg color.Color) string {
	desc := actionDescription(m.pendingAction.Action, m.pendingAction.Target, m.pendingAction.IsGroup)

	style := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextPrimary).
		Background(bg).
		Width(width).
		AlignHorizontal(lipgloss.Center)

	return style.Render(m.spinner.View() + " " + desc)
}

func GroupDetailsPanel() tea.Model {
	return GroupDetailsPanelModel{
		componentId: 2,
		spinner:     newSpinner(),
	}
}
