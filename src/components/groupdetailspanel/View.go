package groupdetailspanel

import (
	"fmt"
	"image/color"
	"slices"

	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

func (m Model) memberServices() []types.ServiceConfig {
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
func (m Model) knownGroups() []string {
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

func (m Model) isServiceRunning(serviceName string) bool {
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
func (m Model) containerForService(serviceName string) (apptypes.DockerContainer, bool) {
	for _, container := range m.containers {
		if container.Service == serviceName {
			return container, true
		}
	}

	return apptypes.DockerContainer{}, false
}

// View renders the panel body and hands it to renderPanelFrame. The body
// is always non-empty (empty states draw their own cards). The action
// buttons are the last block of the body, which pins them to the bottom of
// the panel.
func (m Model) View() tea.View {
	body := m.renderBody()
	screen := chrome.PanelFrame("Details", m.titlePill(), m.isFocused, m.panelWidth, m.panelHeight, body)

	return tea.NewView(screen)
}

// titlePill is the selected group's status pill, which rides on the panel's
// title row. Empty while the panel is showing an empty state: there is no
// group whose status it could report.
func (m Model) titlePill() string {
	if m.selectedGroup == "" || len(m.knownGroups()) == 0 {
		return ""
	}

	members := m.memberServices()

	return statusPill(m.runningCount(members), len(members))
}

func (m Model) runningCount(members []types.ServiceConfig) int {
	running := 0

	for _, svc := range members {
		if m.isServiceRunning(svc.Name) {
			running++
		}
	}

	return running
}

// renderBody builds the panel body for the current state: onboarding,
// nothing-selected, or a selected group's header + member table.
func (m Model) renderBody() string {
	bodyWidth := max(1, chrome.PanelBodyWidth(m.panelWidth))
	bodyAvail := max(1, chrome.PanelBodyHeight(m.panelHeight))
	bg := chrome.PanelBg(m.isFocused)

	// No groups exist anywhere yet -> onboarding.
	if len(m.knownGroups()) == 0 {
		return chrome.EmptyCard(bodyWidth, bodyAvail, bg, "Getting started",
			"Groups are Compose profiles: sets of services you run together. Add a `profiles:` key to a service in your compose file to make one.",
			"n", "new group")
	}

	// Groups exist but none is selected.
	if m.selectedGroup == "" {
		return chrome.EmptyCard(bodyWidth, bodyAvail, bg, "Select a group",
			"Pick a group from the list to see its services.",
			"↑/↓", "to browse")
	}

	// A group is selected: header card + member table + actions.
	members := m.memberServices()
	running := m.runningCount(members)
	stopped := len(members) - running

	content := lipgloss.JoinVertical(lipgloss.Left,
		m.groupHeaderCard(m.selectedGroup, running, stopped, len(members), bodyWidth),
		m.renderMemberTable(members, bodyWidth),
	)

	// The footnote is about the actions, so it belongs at the foot of the panel
	// rather than against the table. It used to ride above the action chip row;
	// with the chips gone it is the panel's only standing hint that a stopped
	// group is one keypress from running - see "The panel footer" in
	// docs/DESIGN.md.
	var footerParts []string
	if running == 0 && len(members) > 0 {
		footerParts = append(footerParts,
			lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).Render("Press s to start."))
	}

	// The spinner replaces the hint while an action is pending: the hint is
	// about a key that is disabled for as long as the spinner is up.
	if m.pendingAction != nil {
		footerParts = []string{m.renderPendingAction(bodyWidth, bg)}
	}

	return chrome.PanelBodyWithFooter(bodyWidth, bodyAvail, bg,
		content, lipgloss.JoinVertical(lipgloss.Left, footerParts...))
}

// groupHeaderCard renders the selected group's name and a
// running/stopped/total summary, separated from the table by a thin rule.
// The status pill sits on the panel's title row - see titlePill.
func (m Model) groupHeaderCard(name string, running, stopped, total int, width int) string {
	nameRow := lipgloss.NewStyle().
		Bold(true).
		Foreground(appstyles.Active.TextPrimary).
		Width(width).
		Render(chrome.Truncate(name, width))

	summary := fmt.Sprintf("%d running · %d stopped · %d services", running, stopped, total)
	summaryRow := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextMuted).
		Width(width).
		Render(summary)

	return lipgloss.JoinVertical(lipgloss.Left, nameRow, summaryRow, chrome.PanelRule(width))
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

// renderMemberTable renders the column headers, a separator, and one row per
// member service, at its natural height. The blank rows between it and the
// action row are panelBodyWithActions's job, so the table does not have to
// know how much of the panel is left below it.
func (m Model) renderMemberTable(members []types.ServiceConfig, width int) string {
	cols := computeCols(width)

	parts := []string{renderTableHeader(cols, width), chrome.PanelRule(width)}

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

	return lipgloss.NewStyle().
		Width(width).
		Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func (m Model) renderMemberRow(cols tableCols, width int, svc types.ServiceConfig) string {
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
	name := lipgloss.NewStyle().Foreground(appstyles.Active.TextPrimary).Width(cols.name).Render(chrome.Truncate(svc.Name, cols.name))
	img := lipgloss.NewStyle().Foreground(appstyles.Active.TextMuted).Width(cols.image).Render(chrome.Truncate(image, cols.image))
	st := lipgloss.NewStyle().Foreground(stateColor(state)).Width(cols.state).Render(chrome.Truncate(state, cols.state))
	hl := lipgloss.NewStyle().Foreground(chrome.HealthColor(health)).Width(cols.health).Render(chrome.Truncate(health, cols.health))
	up := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).Width(cols.uptime).Render(chrome.Truncate(uptime, cols.uptime))
	pt := lipgloss.NewStyle().Foreground(appstyles.Active.TextMuted).Width(cols.ports).Render(chrome.Truncate(ports, cols.ports))

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

// renderPendingAction renders a spinner with the action description in place
// of the action buttons while a docker action is in progress.
func (m Model) renderPendingAction(width int, bg color.Color) string {
	desc := chrome.ActionDescription(m.pendingAction.Action, m.pendingAction.Target, m.pendingAction.IsGroup)

	style := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextPrimary).
		Background(bg).
		Width(width).
		AlignHorizontal(lipgloss.Center)

	return style.Render(m.spinner.View() + " " + desc)
}
