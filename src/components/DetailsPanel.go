package components

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/cmds"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/go-units"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/keys"
	"gopkg.in/yaml.v3"
)

// containerForService returns the first container matching the given compose
// service name, or a zero-value DockerContainer and false if none exists.
func (m DetailsPanelModel) containerForService(serviceName string) (apptypes.DockerContainer, bool) {
	for _, c := range m.containers {
		if c.Service == serviceName {
			return c, true
		}
	}
	return apptypes.DockerContainer{}, false
}

type DetailsPanelModel struct {
	service     *types.ServiceConfig
	panelWidth  int
	panelHeight int
	isFocused   bool
	componentId int

	// editing is true while the inline YAML editor is open.
	editing bool
	// editor is the textarea used in inline edit mode.
	editor textarea.Model
	// originalFragment is the fragment the editor started from, used to
	// detect unsaved changes.
	originalFragment []byte
	// validationError is the last YAML syntax error from the live parse.
	validationError string
	// saveError is the last compose-level error from a failed save.
	saveError string

	pendingAction *PendingAction
	spinner       spinner.Model

	// containers is the latest known container list, used to derive the
	// RUNNING/STOPPED status pill in the panel title row.
	containers []apptypes.DockerContainer
}

func (m DetailsPanelModel) Init() tea.Cmd {
	return nil
}

func (m DetailsPanelModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var finalCmds []tea.Cmd

	switch msg := msg.(type) {
	// Both dimensions come from AppModel. Deriving them from WindowSizeMsg
	// here would leave the panel at width 0 whenever the Services page wasn't
	// the active page at resize time.
	case cmds.SetBodyLayoutMsg:
		m.panelWidth = msg.RightWidth
		m.panelHeight = msg.Height
		m.resizeEditor()

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
		m.syncEditorFocus()

	case cmds.SetSelectedServiceMsg:
		service := types.ServiceConfig(msg)
		// If the service changes while editing, abandon the editor. The
		// selection only changes after a successful save or an external
		// reload; either way the disk state is what matters now.
		if m.editing && (m.service == nil || m.service.Name != service.Name) {
			updated, cmd := m.exitEditMode()
			m = updated
			finalCmds = append(finalCmds, cmd)
		}
		m.service = &service

	case cmds.InlineEditReadyMsg:
		if m.service == nil || m.service.Name != msg.ServiceName || msg.Err != nil {
			break
		}
		updated, cmd := m.enterEditMode(msg.Fragment)
		m = updated
		finalCmds = append(finalCmds, cmd)

	case cmds.ServiceSavedMsg:
		if m.service == nil || m.service.Name != msg.ServiceName || !m.editing {
			break
		}
		if msg.Err != nil {
			m.saveError = msg.Err.Error()
		} else {
			m.saveError = ""
			m.validationError = ""
			updated, cmd := m.exitEditMode()
			m = updated
			finalCmds = append(finalCmds, cmd)
		}

	case cmds.ServiceEditedMsg:
		// The $EDITOR path finished. If it succeeded while we were editing,
		// exit the editor so the reloaded state shows; the panel is not in
		// edit mode for normal external edits, so this is a no-op there.
		if m.editing && msg.Err == nil && m.service != nil && msg.ServiceName == m.service.Name {
			updated, cmd := m.exitEditMode()
			m = updated
			finalCmds = append(finalCmds, cmd)
		}

	case cmds.CancelInlineEditMsg:
		updated, cmd := m.exitEditMode()
		m = updated
		finalCmds = append(finalCmds, cmd)

	case cmds.GetRunningContainersMsg:
		if msg.Err == nil {
			m.containers = msg.Containers
		}

	case cmds.GetContainerStatsMsg:
		// Present-but-unenriched still beats stale: a failed stats call sends
		// the containers through without their runtime numbers.
		if msg.Containers != nil {
			m.containers = msg.Containers
		}

	// A terminal paste arrives as its own message, not as key presses. It only
	// means anything to the editor, and only while the editor is open: the
	// panel's read-only mode has nothing to paste into.
	case tea.PasteMsg:
		var editorCmd tea.Cmd
		m, editorCmd = m.forwardToEditor(msg)
		finalCmds = append(finalCmds, editorCmd)

	case tea.KeyPressMsg:
		if !m.isFocused || m.service == nil {
			break
		}

		if m.editing {
			updated, cmd, handled := m.handleEditKey(msg)
			m = updated
			if handled {
				m.updateValidationError()
				if cmd != nil {
					finalCmds = append(finalCmds, cmd)
				}
				break
			}

			// Not a special edit key: pass it to the textarea and validate.
			var editorCmd tea.Cmd
			m.editor, editorCmd = m.editor.Update(msg)
			finalCmds = append(finalCmds, editorCmd)
			m.updateValidationError()
		} else if action, ok := dockerActionFor(msg); ok {
			finalCmds = append(finalCmds, cmds.RequestDockerAction(action, m.service.Name, false))
		} else if key.Matches(msg, keys.Details.Remove) {
			finalCmds = append(finalCmds, cmds.OpenConfirmModal(
				fmt.Sprintf("Remove service %q?\nThis stops and removes its containers.", m.service.Name),
				cmds.RequestDockerAction("remove", m.service.Name, false),
			))
		} else if key.Matches(msg, keys.Details.Logs) {
			finalCmds = append(finalCmds, cmds.OpenLogsModal(m.service.Name, false))
		} else if key.Matches(msg, keys.Details.EditService) {
			finalCmds = append(finalCmds, cmds.RequestInlineEdit(m.service.Name))
		} else if key.Matches(msg, keys.Details.EditFile) {
			finalCmds = append(finalCmds, cmds.OpenEditor())
		}

	// The textarea's clipboard round trip (ctrl+v -> Paste cmd -> an unexported
	// pasteMsg) comes back as a message this switch cannot name, so anything
	// unrecognised goes to the editor while it is open. Safe by construction:
	// every message the panel acts on has its own case above, so nothing that
	// reaches here was ever the panel's to handle.
	default:
		var editorCmd tea.Cmd
		m, editorCmd = m.forwardToEditor(msg)
		finalCmds = append(finalCmds, editorCmd)
	}

	return m, tea.Batch(finalCmds...)
}

// forwardToEditor passes a message to the open editor and re-validates. It
// is the path for everything the editor answers that is not a key press.
func (m DetailsPanelModel) forwardToEditor(msg tea.Msg) (DetailsPanelModel, tea.Cmd) {
	if !m.editing {
		return m, nil
	}

	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	m.updateValidationError()

	return m, cmd
}

// handleEditKey answers the keys the editor owns: the control keys (save,
// open in $EDITOR, cancel) and the ones that edit the buffer through the
// indent policy rather than as plain text. handled reports whether the key
// was the editor's; when it is false the caller passes the key to the
// textarea as ordinary input.
//
// handled is a separate return rather than "cmd != nil" because Enter is
// handled and produces no command - the buffer edit happens here, in place.
func (m DetailsPanelModel) handleEditKey(msg tea.KeyPressMsg) (DetailsPanelModel, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, keys.Details.Save):
		return m, cmds.RequestSaveService(m.service.Name, []byte(m.editor.Value())), true

	case key.Matches(msg, keys.Details.OpenEditor):
		return m, cmds.OpenServiceEditor(m.service.Name), true

	case key.Matches(msg, keys.Editor.NewLine):
		m.editor.InsertString("\n" + indentAfter(m.currentLine(), m.editor.Column()))
		return m, nil, true

	case key.Matches(msg, keys.Global.Back):
		if m.hasChanges() {
			return m, cmds.OpenConfirmModal(
				"Discard changes?",
				cmds.CancelInlineEdit(),
			), true
		}
		updated, cmd := m.exitEditMode()
		return updated, cmd, true
	}

	return m, nil, false
}

// currentLine is the logical line the cursor is on. The textarea soft-wraps,
// so this is the row in the value, not the row on screen.
func (m DetailsPanelModel) currentLine() string {
	lines := strings.Split(m.editor.Value(), "\n")
	row := m.editor.Line()
	if row < 0 || row >= len(lines) {
		return ""
	}
	return lines[row]
}

// hasChanges reports whether the editor's contents differ from the fragment
// it started with.
func (m DetailsPanelModel) hasChanges() bool {
	return string(m.originalFragment) != m.editor.Value()
}

// enterEditMode sets up the textarea with the given fragment and broadcasts
// that the editor now owns the keyboard.
func (m DetailsPanelModel) enterEditMode(fragment []byte) (DetailsPanelModel, tea.Cmd) {
	m.editing = true
	m.originalFragment = fragment
	m.saveError = ""
	m.validationError = ""

	m.editor = textarea.New()
	m.editor.ShowLineNumbers = false
	m.editor.Prompt = ""
	m.editor.SetValue(string(fragment))
	m.editor.Focus()
	m.resizeEditor()
	m.updateValidationError()

	return m, cmds.SetEditingState(true)
}

// exitEditMode tears down the editor and tells the app the keyboard is free.
func (m DetailsPanelModel) exitEditMode() (DetailsPanelModel, tea.Cmd) {
	m.editing = false
	m.originalFragment = nil
	m.saveError = ""
	m.validationError = ""

	return m, cmds.SetEditingState(false)
}

// syncEditorFocus keeps the textarea focus in sync with the panel focus.
func (m *DetailsPanelModel) syncEditorFocus() {
	if !m.editing {
		return
	}

	if m.isFocused {
		m.editor.Focus()
	} else {
		m.editor.Blur()
	}
}

// resizeEditor constrains the textarea to the panel body. It is called on
// layout changes and on entering edit mode.
func (m *DetailsPanelModel) resizeEditor() {
	if !m.editing {
		return
	}

	bodyW := max(1, panelBodyWidth(m.panelWidth))
	// Reserve two rows for the status line under the editor.
	bodyH := max(1, panelBodyHeight(m.panelHeight)-2)

	m.editor.SetWidth(bodyW)
	m.editor.SetHeight(bodyH)
}

// updateValidationError parses the editor contents as YAML and records the
// error. Full compose validation is deliberately deferred to save.
func (m *DetailsPanelModel) updateValidationError() {
	if !m.editing {
		return
	}

	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(m.editor.Value()), &doc); err != nil {
		m.validationError = err.Error()
		return
	}
	m.validationError = ""
}

// EditorValue returns the editor's current contents. Exported for the model
// tests, which drive paste and indentation through the whole message path and
// need to see what landed in the buffer.
func (m DetailsPanelModel) EditorValue() string {
	return m.editor.Value()
}

// OwnsKeyboard reports whether the panel is holding the whole keyboard. This
// is the same contract the filtered lists use: while true, AppModel stands
// down from its own keys so letters are letters, not commands. The editor
// is always focused when it is open, so it does not need to check focus.
func (m DetailsPanelModel) OwnsKeyboard() bool {
	return m.editing
}

func (m DetailsPanelModel) View() tea.View {
	bodyWidth := max(1, panelBodyWidth(m.panelWidth))
	bodyAvail := max(1, panelBodyHeight(m.panelHeight))

	if m.service == nil {
		body := renderEmptyCard(bodyWidth, bodyAvail, panelBg(m.isFocused), "Select a service",
			"Pick a service from the list to see its details.",
			"↑/↓", "to browse")
		screen := renderPanelFrame("Details", "", m.isFocused, m.panelWidth, m.panelHeight, body)
		return tea.NewView(screen)
	}

	if m.editing {
		body := m.renderEditor(bodyWidth, bodyAvail)
		screen := renderPanelFrame("Edit service", m.service.Name, m.isFocused, m.panelWidth, m.panelHeight, body)
		return tea.NewView(screen)
	}

	bg := panelBg(m.isFocused)

	serviceHeader := m.renderServiceHeader(bodyWidth)
	configTable := m.renderConfigTable(bodyWidth)
	runtimeStats := m.renderRuntimeStats(bodyWidth)

	var buttons string
	if m.pendingAction != nil {
		buttons = m.renderPendingAction(bodyWidth, bg)
	} else {
		buttons = renderActionButtons(bodyWidth, bg)
	}

	parts := []string{serviceHeader, configTable}
	if runtimeStats != "" {
		// Separate the stats table from the config table with a blank
		// line and a rule, matching the visual separation between the
		// service header and the config table from renderServiceHeader.
		sepRule := lipgloss.NewStyle().
			Foreground(appstyles.Active.BorderDefault).
			Width(bodyWidth).
			Render(strings.Repeat("─", max(bodyWidth, 0)))
		parts = append(parts, "", sepRule, runtimeStats)
	}
	parts = append(parts, buttons)

	body := lipgloss.JoinVertical(lipgloss.Left, parts...)
	body = lipgloss.NewStyle().MaxHeight(bodyAvail).Render(body)

	screen := renderPanelFrame("Details", m.titlePill(), m.isFocused, m.panelWidth, m.panelHeight, body)
	return tea.NewView(screen)
}

// titlePill returns a status pill for the selected service, or "" when no
// service is selected. The pill is rendered in the panel's title row,
// right-aligned — same visual language as GroupDetailsPanelModel.titlePill.
func (m DetailsPanelModel) titlePill() string {
	if m.service == nil {
		return ""
	}

	var label string
	var bg, fg color.Color

	if m.isServiceRunning(m.service.Name) {
		label, bg, fg = "RUNNING", appstyles.Active.StatusRunning, appstyles.Active.InkOnLight
	} else {
		label, bg, fg = "STOPPED", appstyles.Active.StatusError, appstyles.Active.InkOnDark
	}

	return lipgloss.NewStyle().
		Background(bg).
		Foreground(fg).
		Bold(true).
		Padding(0, 1).
		Render(label)
}

// isServiceRunning checks whether a live container exists for the given
// compose service name and is in the "running" state.
func (m DetailsPanelModel) isServiceRunning(serviceName string) bool {
	for _, container := range m.containers {
		if container.Service == serviceName && container.State == "running" {
			return true
		}
	}

	return false
}

// renderServiceHeader renders the service name, image, and a status line
// (status dot · state · health · uptime). It mirrors the visual weight of
// GroupDetailsPanelModel.groupHeaderCard.
func (m DetailsPanelModel) renderServiceHeader(width int) string {
	name := m.service.Name
	image := m.service.Image

	nameRow := lipgloss.NewStyle().
		Bold(true).
		Foreground(appstyles.Active.TextPrimary).
		Width(width).
		Render(name)

	var imagePart string
	if image != "" {
		imagePart = " (" + image + ")"
	}
	subtitleRow := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextMuted).
		Width(width).
		Render(imagePart)

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

	if hasContainer && container.HealthStatus != "" && container.HealthStatus != "-" {
		hlColor := healthColor(container.HealthStatus)
		hl := lipgloss.NewStyle().Foreground(hlColor).Render("· " + container.HealthStatus)
		statusParts = append(statusParts, "  ", hl)
	}

	if hasContainer && container.RunningFor != "" {
		up := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).Render("· " + container.RunningFor)
		statusParts = append(statusParts, "  ", up)
	}

	statusRow := lipgloss.NewStyle().
		Width(width).
		Render(lipgloss.JoinHorizontal(lipgloss.Left, statusParts...))

	rule := lipgloss.NewStyle().
		Foreground(appstyles.Active.BorderDefault).
		Width(width).
		Render(strings.Repeat("─", max(width, 0)))

	return lipgloss.JoinVertical(lipgloss.Left, nameRow, subtitleRow, statusRow, "", rule)
}

// renderConfigTable renders the service's compose configuration as a compact
// two-column table (Property | Value), matching the visual language of
// GroupDetailsPanelModel.renderMemberTable.
func (m DetailsPanelModel) renderConfigTable(width int) string {
	propWidth := 14
	valWidth := width - propWidth
	if valWidth < 8 {
		valWidth = 8
		propWidth = width - valWidth
		if propWidth < 4 {
			propWidth = 4
		}
	}

	dim := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).Bold(true)

	headerRow := lipgloss.JoinHorizontal(lipgloss.Left,
		dim.Width(propWidth).Render("PROPERTY"),
		dim.Width(valWidth).Render("VALUE"),
	)

	rule := lipgloss.NewStyle().
		Foreground(appstyles.Active.BorderDefault).
		Width(width).
		Render(strings.Repeat("─", max(width, 0)))

	var rows []string

	svc := *m.service

	// Ports
	if len(svc.Ports) > 0 {
		var portLines []string
		for _, port := range svc.Ports {
			protocol := port.Protocol
			if protocol == "" {
				protocol = "tcp"
			}
			portStr := fmt.Sprintf("%d/%s", port.Target, protocol)
			if port.Published != "" {
				portStr = port.Published + "->" + portStr
			}
			portLines = append(portLines, portStr)
		}
		rows = append(rows, m.configRow(propWidth, valWidth, "Ports", portLines...))
	}

	// Container name
	if svc.ContainerName != "" {
		rows = append(rows, m.configRow(propWidth, valWidth, "Container", svc.ContainerName))
	}

	// Restart policy
	if svc.Restart != "" {
		rows = append(rows, m.configRow(propWidth, valWidth, "Restart", svc.Restart))
	}

	// Networks
	if len(svc.Networks) > 0 {
		netNames := make([]string, 0, len(svc.Networks))
		for name := range svc.Networks {
			netNames = append(netNames, name)
		}
		rows = append(rows, m.configRow(propWidth, valWidth, "Networks", strings.Join(netNames, ", ")))
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
		rows = append(rows, m.configRow(propWidth, valWidth, "Volumes", strings.Join(volParts, ", ")))
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
			rows = append(rows, m.configRow(propWidth, valWidth, "Healthcheck", test))
		}
	}

	// Depends on
	if len(svc.DependsOn) > 0 {
		deps := make([]string, 0, len(svc.DependsOn))
		for name := range svc.DependsOn {
			deps = append(deps, name)
		}
		rows = append(rows, m.configRow(propWidth, valWidth, "Depends on", strings.Join(deps, ", ")))
	}

	// Pull policy
	if svc.PullPolicy != "" {
		rows = append(rows, m.configRow(propWidth, valWidth, "Pull", svc.PullPolicy))
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
		rows = append(rows, m.configRow(propWidth, valWidth, "IDs", strings.Join(idParts, "  ")))
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
		rows = append(rows, m.configRow(propWidth, valWidth, "Memory", strings.Join(memLimits, ", ")))
	}

	// Labels (useful for reverse-proxy configs, etc.)
	if len(svc.Labels) > 0 {
		rows = append(rows, m.configRow(propWidth, valWidth, "Labels", fmt.Sprintf("%d keys", len(svc.Labels))))
	}

	if len(rows) == 0 {
		return ""
	}

	all := make([]string, 0, 2+len(rows))
	all = append(all, headerRow, rule)
	all = append(all, rows...)

	return lipgloss.JoinVertical(lipgloss.Left, all...)
}

// configRow builds one row of the config table: a left-aligned property
// name in muted text and a right-filling value in primary text. When value
// lines span multiple entries, the property label row-spans them visually
// by appearing only on the first line.
func (m DetailsPanelModel) configRow(propWidth, valWidth int, prop string, values ...string) string {
	propStyle := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextDim).
		Width(propWidth).
		Render(prop)

	if len(values) == 0 {
		return lipgloss.JoinHorizontal(lipgloss.Left, propStyle, lipgloss.NewStyle().Width(valWidth).Render("—"))
	}

	valStyle := lipgloss.NewStyle().Foreground(appstyles.Active.TextPrimary).Width(valWidth)

	// First value on the same row as the property.
	firstVal := valStyle.Render(values[0])
	row := lipgloss.JoinHorizontal(lipgloss.Left, propStyle, firstVal)

	// Subsequent values on continuation rows (blank property cell).
	for _, v := range values[1:] {
		cont := lipgloss.JoinHorizontal(lipgloss.Left,
			lipgloss.NewStyle().Width(propWidth).Render(""),
			valStyle.Render(v),
		)
		row = lipgloss.JoinVertical(lipgloss.Left, row, cont)
	}

	return row
}

// renderRuntimeStats renders a card with live container stats (memory,
// CPU, network I/O, disk I/O, PIDs, uptime) as a compact two-column table.
// Returns an empty string when the service has no running container or no
// stats data.
func (m DetailsPanelModel) renderRuntimeStats(width int) string {
	if m.service == nil {
		return ""
	}

	container, ok := m.containerForService(m.service.Name)
	if !ok || container.State != "running" {
		return ""
	}

	// No up-front "do we have stats" check: it listed four of the six fields
	// this renders, so a container reporting only PIDs or only an uptime got
	// nothing. Every row below is already conditional and the len(rows) == 0
	// check at the end is the same guard derived from what actually rendered,
	// so it cannot fall out of step with the rows again.

	propWidth := 14
	valWidth := width - propWidth
	if valWidth < 8 {
		valWidth = 8
		propWidth = width - valWidth
		if propWidth < 4 {
			propWidth = 4
		}
	}

	dim := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).Bold(true)

	headerRow := lipgloss.JoinHorizontal(lipgloss.Left,
		dim.Width(propWidth).Render("METRIC"),
		dim.Width(valWidth).Render("VALUE"),
	)

	rule := lipgloss.NewStyle().
		Foreground(appstyles.Active.BorderDefault).
		Width(width).
		Render(strings.Repeat("─", max(width, 0)))

	var rows []string
	valStyle := lipgloss.NewStyle().Foreground(appstyles.Active.TextPrimary).Width(valWidth)
	propStyle := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).Width(propWidth)

	if container.MemUsage != "" {
		memDisplay := apptypes.FormatMemUsage(container.MemUsage, container.MemPerc)
		row := lipgloss.JoinHorizontal(lipgloss.Left,
			propStyle.Render("Memory"),
			valStyle.Render(memDisplay),
		)
		rows = append(rows, row)
	}

	if container.CPUPerc != "" {
		row := lipgloss.JoinHorizontal(lipgloss.Left,
			propStyle.Render("CPU"),
			valStyle.Render(container.CPUPerc),
		)
		rows = append(rows, row)
	}

	if container.NetIO != "" {
		row := lipgloss.JoinHorizontal(lipgloss.Left,
			propStyle.Render("Network I/O"),
			valStyle.Render(container.NetIO),
		)
		rows = append(rows, row)
	}

	if container.BlockIO != "" {
		row := lipgloss.JoinHorizontal(lipgloss.Left,
			propStyle.Render("Disk I/O"),
			valStyle.Render(container.BlockIO),
		)
		rows = append(rows, row)
	}

	if container.PIDs != "" {
		row := lipgloss.JoinHorizontal(lipgloss.Left,
			propStyle.Render("PIDs"),
			valStyle.Render(container.PIDs),
		)
		rows = append(rows, row)
	}

	if container.RunningFor != "" {
		row := lipgloss.JoinHorizontal(lipgloss.Left,
			propStyle.Render("Uptime"),
			valStyle.Render(container.RunningFor),
		)
		rows = append(rows, row)
	}

	if len(rows) == 0 {
		return ""
	}

	all := make([]string, 0, 2+len(rows))
	all = append(all, headerRow, rule)
	all = append(all, rows...)

	return lipgloss.JoinVertical(lipgloss.Left, all...)
}

// renderEditor renders the textarea plus a status line under it. The status
// line shows live YAML validation, the last save error, and the editor keys.
func (m DetailsPanelModel) renderEditor(bodyWidth, bodyAvail int) string {
	bg := panelBg(m.isFocused)
	editorView := m.editor.View()

	// The textarea has no explicit background; seal it to the panel tier so
	// rows shorter than the editor width do not leak the terminal default.
	editorView = appstyles.FillBackground(bg, editorView)

	status := m.renderStatusLine(bodyWidth)

	content := lipgloss.JoinVertical(lipgloss.Left, editorView, status)
	return fitBox(lipgloss.NewStyle().Background(bg), bodyWidth, bodyAvail).Render(content)
}

// renderStatusLine draws the live validation / save error and the editor keys.
func (m DetailsPanelModel) renderStatusLine(width int) string {
	var statusText string
	var statusColor color.Color

	switch {
	case m.saveError != "":
		statusText = m.saveError
		statusColor = appstyles.Active.Danger
	case m.validationError != "":
		statusText = "YAML: " + m.validationError
		statusColor = appstyles.Active.StatusStarting
	default:
		statusText = "YAML ok"
		statusColor = appstyles.Active.StatusRunning
	}

	bg := panelBg(m.isFocused)

	statusStyle := lipgloss.NewStyle().
		Foreground(statusColor).
		Background(bg).
		Width(width).
		MaxWidth(width)

	hints := renderKeyHints([]KeyHint{
		{Key: "ctrl+s", Desc: "save"},
		{Key: "ctrl+o", Desc: "editor"},
		{Key: "esc", Desc: "cancel"},
	}, appstyles.Active.TextDim)

	hintsStyle := lipgloss.NewStyle().
		Background(bg).
		Width(width).
		MaxWidth(width)

	return lipgloss.JoinVertical(lipgloss.Left,
		statusStyle.Render(statusText),
		hintsStyle.Render(hints),
	)
}

// renderPendingAction renders a spinner with the action description in place
// of the action buttons while a docker action is in progress.
func (m DetailsPanelModel) renderPendingAction(width int, bg color.Color) string {
	desc := actionDescription(m.pendingAction.Action, m.pendingAction.Target, m.pendingAction.IsGroup)

	style := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextPrimary).
		Background(bg).
		Width(width).
		AlignHorizontal(lipgloss.Center)

	return style.Render(m.spinner.View() + " " + desc)
}

func DetailsPanel(service *types.ServiceConfig) tea.Model {
	return DetailsPanelModel{
		service:     service,
		componentId: 2,
		spinner:     newSpinner(),
	}
}
