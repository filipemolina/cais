package components

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/go-units"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/constants"
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

	pendingAction *chrome.PendingAction
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
		m.pendingAction = &chrome.PendingAction{Action: msg.Action, Target: msg.Target, IsGroup: msg.IsGroup}
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
		} else if action, ok := chrome.DockerActionFor(msg); ok {
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

	case key.Matches(msg, keys.Editor.Indent):
		// A pure insertion at a known position, so there is no cursor to
		// restore by hand: insert at the start of the line, then move the
		// cursor right by the same width so the text under it does not
		// shift away.
		col := m.editor.Column()
		m.editor.SetCursorColumn(0)
		m.editor.InsertString(yamlIndent)
		m.editor.SetCursorColumn(col + len(yamlIndent))
		return m, nil, true

	case key.Matches(msg, keys.Editor.Outdent):
		m.outdentCurrentLine()
		return m, nil, true

	case msg.Code == tea.KeyBackspace:
		if updated, ok := m.outdentOnBackspace(); ok {
			return updated, nil, true
		}

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

// outdentCurrentLine removes up to one indent level of leading whitespace
// from the current line. Up to, not exactly: a line indented less than a
// full level (or not at all) outdents to zero rather than going negative,
// and at column 0 with no leading whitespace it is a no-op.
func (m *DetailsPanelModel) outdentCurrentLine() {
	row := m.editor.Line()
	col := m.editor.Column()
	runes := []rune(m.currentLine())

	trimmed := len(runes)
	for i, r := range runes {
		if r != ' ' {
			trimmed = i
			break
		}
	}
	removed := min(trimmed, len(yamlIndent))
	if removed == 0 {
		return
	}

	newLine := string(runes[removed:])
	newCol := max(0, col-removed)
	m.replaceLine(row, newLine, newCol)
}

// outdentOnBackspace deletes back to the previous indent stop when
// everything to the left of the cursor on the current line is spaces. It
// reports whether it applied; when it has not, the caller falls through to
// the textarea's ordinary backspace (deleting one character, or at column 0,
// merging with the line above).
func (m DetailsPanelModel) outdentOnBackspace() (DetailsPanelModel, bool) {
	row := m.editor.Line()
	col := m.editor.Column()
	runes := []rune(m.currentLine())

	if col <= 0 || col > len(runes) {
		return m, false
	}
	for _, r := range runes[:col] {
		if r != ' ' {
			return m, false
		}
	}

	newCol := ((col - 1) / len(yamlIndent)) * len(yamlIndent)
	newLine := string(runes[:newCol]) + string(runes[col:])
	m.replaceLine(row, newLine, newCol)

	return m, true
}

// replaceLine rewrites one logical line in place. The textarea has no
// "replace the current line" API, so this rebuilds the whole value; SetValue
// leaves the cursor at the end of the buffer, so the row and column are
// walked back by hand afterward. This looks redundant and is not: without
// it, editing a line in the middle of a fragment throws the cursor to the
// bottom of it.
func (m *DetailsPanelModel) replaceLine(row int, text string, col int) {
	lines := strings.Split(m.editor.Value(), "\n")
	if row < 0 || row >= len(lines) {
		return
	}
	lines[row] = text

	m.editor.SetValue(strings.Join(lines, "\n"))

	m.editor.MoveToBegin()
	for i := 0; i < row; i++ {
		m.editor.CursorDown()
	}
	m.editor.SetCursorColumn(col)
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

	bodyW := max(1, chrome.PanelBodyWidth(m.panelWidth))
	// Reserve two rows for the status line under the editor.
	bodyH := max(1, chrome.PanelBodyHeight(m.panelHeight)-2)

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

// EditorCursor returns the editor's current row and column. Exported for the
// model tests covering tab/outdent/backspace, which pin cursor position, not
// just buffer text - that is where the SetValue-resets-the-cursor bug would
// show up.
func (m DetailsPanelModel) EditorCursor() (int, int) {
	return m.editor.Line(), m.editor.Column()
}

// OwnsKeyboard reports whether the panel is holding the whole keyboard. This
// is the same contract the filtered lists use: while true, AppModel stands
// down from its own keys so letters are letters, not commands. The editor
// is always focused when it is open, so it does not need to check focus.
func (m DetailsPanelModel) OwnsKeyboard() bool {
	return m.editing
}

// actionContext is what the panel knows about the screen, in the shape
// keys.Active reads. The panel is the Services page's right pane, so the page
// and the component id are constants here; the only question is whether this
// pane holds focus, and when it does not, focus is on the list beside it. That
// is deliberately not special-cased: reporting the list as focused makes
// keys.Active return the list's keys, none of which are the action bindings, so
// the buttons dim by the same rule that empties them out of the footer.
func (m DetailsPanelModel) actionContext() keys.Context {
	focused := constants.COMPONENT_BODY_LIST
	if m.isFocused {
		focused = constants.COMPONENT_BODY_DETAILS
	}

	return keys.Context{
		Page:          "Services",
		Focused:       focused,
		Selected:      m.service != nil,
		Editing:       m.editing,
		PendingAction: m.pendingAction != nil,
	}
}

func (m DetailsPanelModel) View() tea.View {
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

	var footer string
	if m.pendingAction != nil {
		footer = m.renderPendingAction(bodyWidth, bg)
	} else {
		footer = chrome.ActionButtons(bodyWidth, bg, m.actionContext())
	}

	body := chrome.PanelBodyWithActions(bodyWidth, bodyAvail, bg,
		lipgloss.JoinVertical(lipgloss.Left, parts...), footer)

	screen := chrome.PanelFrame("Details", m.titlePill(), m.isFocused, m.panelWidth, m.panelHeight, body)
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
// the table has no rows reading "Healthcheck —".
func (m DetailsPanelModel) configRows() []propRow {
	var rows []propRow

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
		rows = append(rows, propRow{"Ports", portLines})
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
func (m DetailsPanelModel) runtimeRows() []propRow {
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
func (m DetailsPanelModel) renderConfigTable(width int) string {
	return renderPropTable("PROPERTY", width, m.configRows())
}

func (m DetailsPanelModel) renderRuntimeStats(width int) string {
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
func (m DetailsPanelModel) renderTables(width int) string {
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
func (m DetailsPanelModel) renderEditor(bodyWidth, bodyAvail int) string {
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
func (m DetailsPanelModel) validationPill() string {
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
func (m DetailsPanelModel) renderEditorHints(width int) string {
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

// renderPendingAction renders a spinner with the action description in place
// of the action buttons while a docker action is in progress.
func (m DetailsPanelModel) renderPendingAction(width int, bg color.Color) string {
	desc := chrome.ActionDescription(m.pendingAction.Action, m.pendingAction.Target, m.pendingAction.IsGroup)

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
		spinner:     chrome.NewSpinner(),
	}
}
