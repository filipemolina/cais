package detailspanel

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/components/chrome"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/docker/go-units"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/keys"
	"github.com/filipemolina/cais/src/utils"
)

// containerForService returns the first container matching the given compose
// service name, or a zero-value DockerContainer and false if none exists.
func (m Model) containerForService(serviceName string) (apptypes.DockerContainer, bool) {
	for _, c := range m.containers {
		if c.Service == serviceName {
			return c, true
		}
	}
	return apptypes.DockerContainer{}, false
}

func (m Model) View() tea.View {
	bodyWidth := max(1, chrome.PanelBodyWidth(m.panelWidth))
	bodyAvail := max(1, chrome.PanelBodyHeight(m.panelHeight))

	if m.service == nil {
		body := chrome.EmptyCard(bodyWidth, bodyAvail, chrome.PanelBg(m.isFocused), "Select a service",
			"Pick a service from the list to see its details.",
			"↑/↓", "to browse")
		screen := chrome.PanelFrame("Details", "", m.isFocused, m.panelWidth, m.panelHeight, body)
		return tea.NewView(screen)
	}

	if m.editing {
		body := m.renderEditor(bodyWidth, bodyAvail)

		// Combine the service name with the live validation status in the
		// title's right-aligned area, so the user sees validation feedback
		// without having to look at the bottom of the editor.
		serviceName := m.service.Name
		validation := m.validationPill()
		right := lipgloss.JoinHorizontal(lipgloss.Left, serviceName+"  ", validation)

		screen := chrome.PanelFrame("Edit service", right, m.isFocused, m.panelWidth, m.panelHeight, body)
		return tea.NewView(screen)
	}

	bg := chrome.PanelBg(m.isFocused)

	parts := []string{m.renderServiceHeader(bodyWidth)}
	if tables := m.renderTables(bodyWidth); tables != "" {
		parts = append(parts, tables)
	}

	// The footer is the pending-action spinner, the apply hint, the y
	// confirmation, or nothing at all. The panel used to pin a row of
	// action chips here when no action was running; the keys it advertised
	// are on the footer bar, and a chip that cannot be clicked is a control
	// that promises more than the panel can do - see "The panel footer" in
	// docs/DESIGN.md. Apply outranks the URL confirmation because it is
	// the newer, more actionable fact - both are rare enough that a real
	// collision is unlikely, but a hint the user still needs to act on
	// should not be silently replaced by one they already have.
	var footer string
	switch {
	case m.pendingAction != nil:
		footer = m.renderPendingAction(bodyWidth, bg)
	case m.applyHint != "":
		footer = lipgloss.NewStyle().
			Foreground(appstyles.Active.StatusStarting).
			Background(bg).
			Width(bodyWidth).
			Render(m.applyHint)
	case m.urlMessage != "":
		footer = lipgloss.NewStyle().
			Foreground(appstyles.Active.TextDim).
			Background(bg).
			Width(bodyWidth).
			Render(m.urlMessage)
	}

	body := chrome.PanelBodyWithFooter(bodyWidth, bodyAvail, bg,
		lipgloss.JoinVertical(lipgloss.Left, parts...), footer)

	screen := chrome.PanelFrame("Details", m.titlePill(), m.isFocused, m.panelWidth, m.panelHeight, body)
	return tea.NewView(screen)
}

// titlePill returns a status pill for the selected service, or "" when no
// service is selected. The pill is rendered in the panel's title row,
// right-aligned — same visual language as groupdetailspanel.Model.titlePill.
func (m Model) titlePill() string {
	if m.service == nil {
		return ""
	}

	var label string
	var bg color.Color

	if m.isServiceRunning(m.service.Name) {
		label, bg = "RUNNING", appstyles.Active.StatusRunning
	} else {
		label, bg = "STOPPED", appstyles.Active.StatusError
	}
	fg := appstyles.InkOn(bg)

	return lipgloss.NewStyle().
		Background(bg).
		Foreground(fg).
		Bold(true).
		Padding(0, 1).
		Render(label)
}

// isServiceRunning checks whether a live container exists for the given
// compose service name and is in the "running" state.
func (m Model) isServiceRunning(serviceName string) bool {
	for _, container := range m.containers {
		if container.Service == serviceName && container.State == "running" {
			return true
		}
	}

	return false
}

// renderServiceHeader renders the service name, image, and a status line
// (status dot · state · health · uptime). It mirrors the visual weight of
// groupdetailspanel.Model.groupHeaderCard.
func (m Model) renderServiceHeader(width int) string {
	name := m.service.Name
	image := m.service.Image

	nameRow := lipgloss.NewStyle().
		Bold(true).
		Foreground(appstyles.Active.TextPrimary).
		Width(width).
		Render(chrome.Truncate(name, width))

	// The image is set flush left in muted text, the same shape the group
	// header's summary line has. It used to be parenthesised and indented by a
	// space, which was the only line in either panel that did not start on the
	// body's left edge.
	subtitleRow := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextMuted).
		Width(width).
		Render(chrome.Truncate(image, width))

	// Status line with dot, state, health, uptime.
	var statusParts []string

	container, hasContainer := m.containerForService(name)
	dotColor := appstyles.Active.StatusStopped
	stateLabel := "stopped"
	if hasContainer {
		stateLabel = container.State
	}
	if m.isServiceRunning(name) {
		dotColor = appstyles.Active.StatusRunning
	}

	dot := lipgloss.NewStyle().Foreground(dotColor).Render("●")
	statusParts = append(statusParts, dot, " ", stateLabel)

	// " · " between the facts, the separator the group header's summary line
	// uses. The double space this had before made the same line read as two.
	sep := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).Render(" · ")

	if hasContainer && container.HealthStatus != "" && container.HealthStatus != "-" {
		hl := lipgloss.NewStyle().Foreground(chrome.HealthColor(container.HealthStatus)).Render(container.HealthStatus)
		statusParts = append(statusParts, sep, hl)
	}

	if hasContainer && container.RunningFor != "" {
		up := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).Render(container.RunningFor)
		statusParts = append(statusParts, sep, up)
	}

	statusRow := lipgloss.NewStyle().
		Width(width).
		Render(lipgloss.JoinHorizontal(lipgloss.Left, statusParts...))

	// No blank row before the rule: the group's header card closes on the rule
	// directly, and the two headers have to read as the same component.
	return lipgloss.JoinVertical(lipgloss.Left, nameRow, subtitleRow, statusRow, chrome.PanelRule(width))
}

// propRow is one row of a labelled two-column table: the property name and
// the value lines under it. Values are a slice because a property like ports
// can carry several, each on its own continuation row.
type propRow struct {
	label  string
	values []string
}

// propTableCols splits a table's width into the label and value columns.
func propTableCols(width int) (int, int) {
	propWidth := 14
	valWidth := width - propWidth
	if valWidth < 8 {
		valWidth = 8
		propWidth = max(4, width-valWidth)
	}

	return propWidth, valWidth
}

// renderPropTable renders a two-column table - a dim bold heading row, a rule
// under it, then one row per entry - in the same visual language as the group
// panel's member table. Both of the service panel's tables are this function;
// they differ only in their heading and their rows.
//
// Returns "" for an empty row set, so a caller can drop the whole section
// rather than leave a heading with nothing under it.
func renderPropTable(heading string, width int, rows []propRow) string {
	if len(rows) == 0 {
		return ""
	}

	propWidth, valWidth := propTableCols(width)
	dim := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).Bold(true)

	lines := make([]string, 0, 2+len(rows))
	lines = append(lines,
		lipgloss.JoinHorizontal(lipgloss.Left,
			dim.Width(propWidth).Render(heading),
			dim.Width(valWidth).Render("VALUE"),
		),
		chrome.PanelRule(width),
	)

	for _, row := range rows {
		lines = append(lines, renderPropRow(propWidth, valWidth, row))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderPropRow builds one row: a left-aligned property name in dim text and a
// right-filling value in primary text. When a property has several values, the
// label row-spans them visually by appearing only on the first line.
//
// Values are truncated to their column, as the member table's cells are: a
// value wider than the column would otherwise wrap onto a line with no label
// beside it, which reads as a continuation row that isn't one.
func renderPropRow(propWidth, valWidth int, row propRow) string {
	propStyle := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextDim).
		Width(propWidth).
		Render(chrome.Truncate(row.label, propWidth))

	if len(row.values) == 0 {
		return lipgloss.JoinHorizontal(lipgloss.Left, propStyle, lipgloss.NewStyle().Width(valWidth).Render("—"))
	}

	valStyle := lipgloss.NewStyle().Foreground(appstyles.Active.TextPrimary).Width(valWidth)

	// First value on the same row as the property.
	out := lipgloss.JoinHorizontal(lipgloss.Left, propStyle, valStyle.Render(chrome.Truncate(row.values[0], valWidth)))

	// Subsequent values on continuation rows (blank property cell).
	for _, v := range row.values[1:] {
		cont := lipgloss.JoinHorizontal(lipgloss.Left,
			lipgloss.NewStyle().Width(propWidth).Render(""),
			valStyle.Render(chrome.Truncate(v, valWidth)),
		)
		out = lipgloss.JoinVertical(lipgloss.Left, out, cont)
	}

	return out
}

// configRows collects the compose configuration the panel reports, in display
// order. Every entry is conditional: a service states only what it defines, so
// the table has no rows reading "Healthcheck —". valWidth is the value
// column's width, needed only for the Web row: chrome.Hyperlink's OSC 8
// sequence must wrap text that is already truncated and padded to its
// column, or chrome.Truncate (runewidth, not ANSI-aware) will cut through
// the middle of the escape sequence later and corrupt the rest of the
// screen - see docs/plans/service-urls.md D5.
func (m Model) configRows(valWidth int) []propRow {
	var rows []propRow

	svc := *m.service

	// Ports
	if len(svc.Ports) > 0 {
		var portLines []string
		for _, port := range svc.Ports {
			portLines = append(portLines, chrome.PortLabel(port))
		}
		rows = append(rows, propRow{"Ports", portLines})
	}

	// Web - the same fact as Ports, one altitude up: an address rather than
	// a mapping. Directly under Ports for that reason.
	if u, ok := utils.ResolveURL(svc, m.host); ok {
		text := chrome.Truncate(u.URL, valWidth)
		rows = append(rows, propRow{"Web", []string{chrome.Hyperlink(text, u.URL)}})
	}

	// Container name
	if svc.ContainerName != "" {
		rows = append(rows, propRow{"Container", []string{svc.ContainerName}})
	}

	// Restart policy
	if svc.Restart != "" {
		rows = append(rows, propRow{"Restart", []string{svc.Restart}})
	}

	// Networks
	if len(svc.Networks) > 0 {
		netNames := make([]string, 0, len(svc.Networks))
		for name := range svc.Networks {
			netNames = append(netNames, name)
		}
		rows = append(rows, propRow{"Networks", []string{strings.Join(netNames, ", ")}})
	}

	// Volumes summary
	if len(svc.Volumes) > 0 {
		bindCount := 0
		volCount := 0
		for _, vol := range svc.Volumes {
			switch vol.Type {
			case "bind", "":
				bindCount++
			case "volume":
				volCount++
			}
		}
		var volParts []string
		if bindCount > 0 {
			volParts = append(volParts, fmt.Sprintf("%d bind", bindCount))
		}
		if volCount > 0 {
			volParts = append(volParts, fmt.Sprintf("%d volume", volCount))
		}
		rows = append(rows, propRow{"Volumes", []string{strings.Join(volParts, ", ")}})
	}

	// Healthcheck
	if svc.HealthCheck != nil && svc.HealthCheck.Test != nil && len(svc.HealthCheck.Test) > 0 {
		test := strings.Join(svc.HealthCheck.Test, " ")
		// Trim common prefixes for brevity.
		test = strings.TrimPrefix(test, "CMD-SHELL ")
		test = strings.TrimPrefix(test, "CMD ")
		test = strings.TrimPrefix(test, "NONE")
		if len(test) > 24 {
			test = test[:24] + "…"
		}
		if test != "" {
			rows = append(rows, propRow{"Healthcheck", []string{test}})
		}
	}

	// Depends on
	if len(svc.DependsOn) > 0 {
		deps := make([]string, 0, len(svc.DependsOn))
		for name := range svc.DependsOn {
			deps = append(deps, name)
		}
		rows = append(rows, propRow{"Depends on", []string{strings.Join(deps, ", ")}})
	}

	// Pull policy
	if svc.PullPolicy != "" {
		rows = append(rows, propRow{"Pull", []string{svc.PullPolicy}})
	}

	// PUID / PGID (common in self-hosted stacks)
	puid, puidOk := svc.Environment["PUID"]
	pgid, pgidOk := svc.Environment["PGID"]
	if puidOk || pgidOk {
		var idParts []string
		if puidOk && puid != nil {
			idParts = append(idParts, "PUID="+*puid)
		}
		if pgidOk && pgid != nil {
			idParts = append(idParts, "PGID="+*pgid)
		}
		rows = append(rows, propRow{"IDs", []string{strings.Join(idParts, "  ")}})
	}

	// Memory limits
	var memLimits []string
	if svc.MemLimit > 0 {
		memLimits = append(memLimits, "limit="+units.BytesSize(float64(svc.MemLimit)))
	}
	if svc.MemReservation > 0 {
		memLimits = append(memLimits, "reservation="+units.BytesSize(float64(svc.MemReservation)))
	}
	if len(memLimits) > 0 {
		rows = append(rows, propRow{"Memory", []string{strings.Join(memLimits, ", ")}})
	}

	// Labels (useful for reverse-proxy configs, etc.)
	if len(svc.Labels) > 0 {
		rows = append(rows, propRow{"Labels", []string{fmt.Sprintf("%d keys", len(svc.Labels))}})
	}

	return rows
}

// runtimeRows collects the live container stats the panel reports. Empty when
// the service has no running container.
//
// No up-front "do we have stats" check: it listed four of the six fields this
// gathers, so a container reporting only PIDs or only an uptime got nothing.
// Every entry below is already conditional, so the caller's "no rows, no
// table" rule is derived from what there actually is to show.
func (m Model) runtimeRows() []propRow {
	if m.service == nil {
		return nil
	}

	container, ok := m.containerForService(m.service.Name)
	if !ok || container.State != "running" {
		return nil
	}

	var rows []propRow

	if container.MemUsage != "" {
		rows = append(rows, propRow{"Memory", []string{apptypes.FormatMemUsage(container.MemUsage, container.MemPerc)}})
	}

	if container.CPUPerc != "" {
		rows = append(rows, propRow{"CPU", []string{container.CPUPerc}})
	}

	if container.NetIO != "" {
		rows = append(rows, propRow{"Network I/O", []string{container.NetIO}})
	}

	if container.BlockIO != "" {
		rows = append(rows, propRow{"Disk I/O", []string{container.BlockIO}})
	}

	if container.PIDs != "" {
		rows = append(rows, propRow{"PIDs", []string{container.PIDs}})
	}

	if container.RunningFor != "" {
		rows = append(rows, propRow{"Uptime", []string{container.RunningFor}})
	}

	return rows
}

// renderConfigTable and renderRuntimeStats are the two tables the panel shows,
// at the width they are given.
func (m Model) renderConfigTable(width int) string {
	_, valWidth := propTableCols(width)
	return renderPropTable("PROPERTY", width, m.configRows(valWidth))
}

func (m Model) renderRuntimeStats(width int) string {
	return renderPropTable("METRIC", width, m.runtimeRows())
}

// tablesMinSideBySide is the body width from which the config and runtime
// tables sit next to each other instead of stacking. Below it each column
// would be too narrow to hold a value like "PUID=1002  PGID=1001" without
// truncating it, and a truncated table is worse than a tall one.
const tablesMinSideBySide = 72

// tablesGutter is the blank column between the two tables. It is what makes
// them read as two tables rather than a four-column one.
const tablesGutter = 3

// renderTables lays the config and runtime tables out for the width available.
//
// Side by side when there is room, because stacked they used a fifth of the
// panel's width and twice its height - a value column of seventy blank columns
// beside a table that had run off the bottom of what the eye takes in at once.
// The group panel keeps one full-width table because it has one table; this is
// the same table style spent on two.
func (m Model) renderTables(width int) string {
	if width >= tablesMinSideBySide {
		colWidth := (width - tablesGutter) / 2

		config := m.renderConfigTable(colWidth)
		stats := m.renderRuntimeStats(colWidth)

		// Only when there are two: a stopped service has no runtime table, and
		// a half-width config table with nothing beside it is just a narrower
		// version of the wasted width this avoids.
		if config != "" && stats != "" {
			return lipgloss.JoinHorizontal(lipgloss.Top,
				config,
				lipgloss.NewStyle().Width(tablesGutter).Render(""),
				stats,
			)
		}
	}

	config := m.renderConfigTable(width)
	stats := m.renderRuntimeStats(width)

	switch {
	case stats == "":
		return config
	case config == "":
		return stats
	}

	// Stacked, the two tables need the same blank-line-and-rule separation the
	// header has from the config table, or they read as one table that changed
	// its mind about its heading halfway down.
	return lipgloss.JoinVertical(lipgloss.Left, config, "", chrome.PanelRule(width), stats)
}

// renderEditor renders the textarea with the editor key hints below it. The
// live YAML validation status is shown in the panel title row instead.
func (m Model) renderEditor(bodyWidth, bodyAvail int) string {
	bg := chrome.PanelBg(m.isFocused)
	editorView := m.editor.View()

	// The textarea has no explicit background; seal it to the panel tier so
	// rows shorter than the editor width do not leak the terminal default.
	editorView = appstyles.FillBackground(bg, editorView)

	hints := m.renderEditorHints(bodyWidth)

	content := lipgloss.JoinVertical(lipgloss.Left, editorView, hints)
	return chrome.FitBox(lipgloss.NewStyle().Background(bg), bodyWidth, bodyAvail).Render(content)
}

// validationPill returns a colored pill for the editor's live YAML validation
// status, suitable for the panel title row right-aligned area. Empty when the
// editor is not open (caller should check m.editing first).
func (m Model) validationPill() string {
	var label string
	var bg color.Color

	switch {
	case m.saveError != "":
		label = m.saveError
		bg = appstyles.Active.Danger
	case m.validationError != "":
		label = "YAML: " + m.validationError
		bg = appstyles.Active.StatusStarting
	default:
		label = "YAML ok"
		bg = appstyles.Active.StatusRunning
	}
	fg := appstyles.InkOn(bg)

	return lipgloss.NewStyle().
		Background(bg).
		Foreground(fg).
		Bold(true).
		Padding(0, 1).
		Render(label)
}

// renderEditorHints renders the editor key hints below the textarea.
func (m Model) renderEditorHints(width int) string {
	bg := chrome.PanelBg(m.isFocused)

	hints := chrome.RenderKeyHints([]chrome.KeyHint{
		chrome.HintFor(keys.Details.Save),
		chrome.HintFor(keys.Details.OpenEditor),
		chrome.HintFor(keys.Editor.Indent),
		chrome.HintFor(keys.Editor.Outdent),
		chrome.HintAs(keys.Global.Back, "cancel"),
	}, appstyles.Active.TextDim)

	return lipgloss.NewStyle().
		Background(bg).
		Width(width).
		MaxWidth(width).
		Render(hints)
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
