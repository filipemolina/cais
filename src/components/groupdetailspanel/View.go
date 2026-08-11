package groupdetailspanel

import (
	"fmt"
	"image/color"
	"slices"
	"strings"

	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/components/chrome"

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
// is always non-empty (empty states draw their own cards). The footer is the
// last block of the body, which pins it to the bottom of the panel.
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

	// No groups exist anywhere yet -> the service overview, or onboarding
	// when there is nothing to show one of.
	if len(m.knownGroups()) == 0 {
		return m.renderServiceOverview(bodyWidth, bodyAvail, bg)
	}

	// Groups exist but none is selected.
	if m.selectedGroup == "" {
		return chrome.EmptyCard(bodyWidth, bodyAvail, bg, "Select a group",
			"Pick a group from the list to see its services.",
			"↑/↓", "to browse")
	}

	// A group is selected: header card + member table.
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

// renderServiceOverview replaces the plain "Getting started" onboarding
// card with a live list of the compose file's services when it has some but
// no groups reference any of them yet (knownGroups() derives every group
// name from services' Profiles, so knownGroups() == 0 means every loaded
// service is already exactly the "ungrouped" set - there is no separate
// filter to apply). The reader gets to see what they have to work with
// before creating the first group, rather than only an explanation of what
// a group is. Falls back to the original onboarding card when the compose
// file has no services at all - a fresh or newly bootstrapped file - where
// there is nothing to list.
func (m Model) renderServiceOverview(width, availHeight int, bg color.Color) string {
	if len(m.services) == 0 {
		return chrome.EmptyCard(width, availHeight, bg, "Getting started",
			"Groups are Compose profiles: sets of services you run together. Add a `profiles:` key to a service in your compose file to make one.",
			"n", "new group")
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(appstyles.Active.TextPrimary).
		Width(width).
		Render(fmt.Sprintf("%d %s, no groups yet", len(m.services), plural(len(m.services), "service")))

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		m.renderMemberTable(m.services, width),
	)

	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(appstyles.Active.Accent)
	hintStyle := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim)
	hint := keyStyle.Render("n") + hintStyle.Render(" create your first group  ") +
		keyStyle.Render("2") + hintStyle.Render(" browse services")

	return chrome.PanelBodyWithFooter(width, availHeight, bg, content, hint)
}

// plural is the naive English plural of word for n: enough for the handful
// of countable nouns this panel puts in front of a number.
func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
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

	// Truncated for the same reason the name above it is: Width() pads but does
	// not clip, so on a narrow panel the summary wrapped onto a second line and
	// cost the member table a row.
	summary := fmt.Sprintf("%d running · %d stopped · %d services", running, stopped, total)
	summaryRow := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextMuted).
		Width(width).
		Render(chrome.Truncate(summary, width))

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
// panel footer are chrome.PanelBodyWithFooter's job, so the table does not have
// to know how much of the panel is left below it.
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
	}

	ports := "—"
	if published := chrome.PublishedPorts(svc.Ports); len(published) > 0 {
		ports = strings.Join(published, ", ")
	}

	if state == "running" {
		dotColor = appstyles.Active.StatusRunning
	}

	// The cell's text and ink per column, walked in columnOrder so the row
	// cannot drift from the header above it. A dropped column has width 0 and
	// is skipped, the same as in renderTableHeader.
	cell := map[string]struct {
		text string
		fg   color.Color
	}{
		"dot":    {"●", dotColor},
		"name":   {svc.Name, appstyles.Active.TextPrimary},
		"image":  {chrome.ShortImage(image, max(1, cols.image-1)), appstyles.Active.TextMuted},
		"state":  {state, stateColor(state)},
		"health": {health, chrome.HealthColor(health)},
		"uptime": {uptime, appstyles.Active.TextDim},
		"ports":  {ports, appstyles.Active.TextMuted},
	}

	var cells []string
	for _, name := range columnOrder {
		w := cols.get(name)
		if w == 0 {
			continue
		}

		// Truncated to one less than the column so there is always a column of
		// gap after it. Truncating to the full width let a long name run flush
		// into the next cell - `navidromedeluan/n…` - which reads as one value
		// rather than two, the same collision the headings had.
		text := cell[name].text
		if name != "dot" {
			text = chrome.Truncate(text, max(1, w-1))
		}

		cells = append(cells, lipgloss.NewStyle().Foreground(cell[name].fg).Width(w).Render(text))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Left, cells...)

	// Clipped for the same reason the header above it is.
	return lipgloss.NewStyle().Width(width).MaxHeight(1).Render(row)
}

func renderTableHeader(cols tableCols, width int) string {
	dim := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).Bold(true)

	// A dropped column has width 0 and is skipped rather than rendered empty:
	// Width(0) does not truncate, so it would print its heading in full over
	// whatever comes next.
	var cells []string
	for _, name := range columnOrder {
		if w := cols.get(name); w > 0 {
			cells = append(cells, dim.Width(w).Render(heading[name]))
		}
	}

	row := lipgloss.JoinHorizontal(lipgloss.Left, cells...)

	// MaxHeight is the backstop under the column dropping: below about seven
	// columns not even the name fits beside the dot, and a wrapped heading row
	// would push the table's own rows down the panel.
	return lipgloss.NewStyle().Width(width).MaxHeight(1).Render(row)
}

func stateColor(state string) color.Color {
	if state == "running" {
		return appstyles.Active.StatusRunning
	}

	return appstyles.Active.StatusStopped
}

// tableCols holds the per-column widths for the member table. A width of 0
// means the column was dropped for want of room - see computeCols - and both
// the header and the rows skip it rather than rendering an empty cell.
type tableCols struct {
	dot, name, image, state, health, uptime, ports int
}

// columnOrder is the left-to-right order of the table's columns. The header,
// the rows and the width arithmetic all walk it, so a column cannot be added
// to one of them and forgotten in another.
var columnOrder = []string{"dot", "name", "image", "state", "health", "uptime", "ports"}

// heading is a column's label, and "" for the status dot, which is its own
// legend. minWidth is derived from it: a column narrower than its own heading
// is what produced NAMEIMAGSTATHEALT... - lipgloss pads to Width but does not
// truncate, so an over-long heading runs into the next column, and at the
// narrowest widths wraps the header onto a second and third line.
var heading = map[string]string{
	"dot": "", "name": "NAME", "image": "IMAGE", "state": "STATE",
	"health": "HEALTH", "uptime": "UPTIME", "ports": "PORTS",
}

// dropOrder is the order columns are given up in when the panel cannot hold
// them all, lowest first. It is deliberately not the display order, the same
// distinction the footer bar's keys.Priority makes:
//
// Ports goes first - it is the widest column and the service details panel
// prints the same information with room to spare. Image next: the row is
// identified by its name, not its tag. Health and uptime are the detail a
// glance does not need. State is the last to go because "is it running?" is
// what the table is for - and even then the status dot answers it in two
// columns, which is why dot and name are absent here: they are the row's
// identity and are never dropped.
var dropOrder = []string{"ports", "image", "health", "uptime", "state"}

func (c tableCols) get(name string) int {
	switch name {
	case "dot":
		return c.dot
	case "name":
		return c.name
	case "image":
		return c.image
	case "state":
		return c.state
	case "health":
		return c.health
	case "uptime":
		return c.uptime
	case "ports":
		return c.ports
	}

	return 0
}

func (c *tableCols) set(name string, width int) {
	switch name {
	case "dot":
		c.dot = width
	case "name":
		c.name = width
	case "image":
		c.image = width
	case "state":
		c.state = width
	case "health":
		c.health = width
	case "uptime":
		c.uptime = width
	case "ports":
		c.ports = width
	}
}

func (c tableCols) total() int {
	sum := 0
	for _, name := range columnOrder {
		sum += c.get(name)
	}

	return sum
}

// minWidth is the narrowest a column can be and still print its own heading
// with a column of gap after it, so two headings never touch.
func minWidth(name string) int {
	if name == "dot" {
		return 2
	}

	return len(heading[name]) + 1
}

// minTotal is what the surviving columns need between them.
func (c tableCols) minTotal() int {
	sum := 0
	for _, name := range columnOrder {
		if c.get(name) > 0 {
			sum += minWidth(name)
		}
	}

	return sum
}

// computeCols distributes the available width across the columns. Wide
// terminals expand the flexible columns (name, image, ports); narrow terminals
// drop whole columns in dropOrder and then shrink what is left, never below the
// width of its own heading.
//
// Dropping rather than shrinking-to-nothing is the same fix the footer bar and
// the details panels' action row got: a fixed set of controls squeezed past
// legibility mangles, where a smaller set of whole ones still reads. Everything
// dropped here is still in the service details panel.
func computeCols(width int) tableCols {
	if width < 1 {
		width = 1
	}

	c := tableCols{
		dot: 2, name: 18, image: 16, state: 9, health: 8, uptime: 11, ports: 16,
	}

	for _, name := range dropOrder {
		if c.minTotal() <= width {
			break
		}
		c.set(name, 0)
	}

	// Shrink the widest column that still has room to give, until the row fits.
	for c.total() > width {
		widest := widestShrinkable(c)
		if widest == "" {
			break
		}
		c.set(widest, c.get(widest)-1)
	}

	// Expand the flexible columns to fill wide terminals, skipping any that
	// were dropped - a panel with no ports column gives that share to the name.
	if extra := width - c.total(); extra > 0 {
		var flexible []string
		for _, name := range []string{"ports", "name", "image"} {
			if c.get(name) > 0 {
				flexible = append(flexible, name)
			}
		}

		for i, name := range flexible {
			share := extra / len(flexible)
			if i == len(flexible)-1 {
				// The last one absorbs the rounding, so the row fills the width
				// exactly rather than leaving a column or two unpainted.
				share = extra - share*(len(flexible)-1)
			}
			c.set(name, c.get(name)+share)
		}
	}

	return c
}

// widestShrinkable is the column with the most to give: the widest one still
// above its own minimum, or "" when every surviving column is at its floor.
func widestShrinkable(c tableCols) string {
	widest, most := "", 0

	for _, name := range columnOrder {
		width := c.get(name)
		if width > minWidth(name) && width > most {
			widest, most = name, width
		}
	}

	return widest
}

// renderPendingAction renders a spinner with the action description in the
// panel's footer while a docker action is in progress.
func (m Model) renderPendingAction(width int, bg color.Color) string {
	desc := chrome.ActionDescription(m.pendingAction.Action, m.pendingAction.Target, m.pendingAction.IsGroup)

	style := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextPrimary).
		Background(bg).
		Width(width).
		AlignHorizontal(lipgloss.Center)

	return style.Render(m.spinner.View() + " " + desc)
}
