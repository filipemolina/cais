package components

import (
	"fmt"
	"image/color"

	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/cmds"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/compose-spec/compose-go/v2/types"
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
		if msg.Err == nil {
			m.containers = msg.Containers
		}

	case tea.KeyPressMsg:
		if !m.isFocused || m.service == nil {
			break
		}

		if m.editing {
			updated, cmd := m.handleEditKey(msg)
			if cmd != nil {
				return updated, cmd
			}
			m = updated

			// Not a special edit key: pass it to the textarea and validate.
			var editorCmd tea.Cmd
			m.editor, editorCmd = m.editor.Update(msg)
			finalCmds = append(finalCmds, editorCmd)
			m.updateValidationError()
		} else if action, ok := dockerActionFor(msg); ok {
			finalCmds = append(finalCmds, cmds.RequestDockerAction(action, m.service.Name, false))
		} else if key.Matches(msg, keys.Details.Remove) {
			finalCmds = append(finalCmds, cmds.OpenConfirmModal(
				fmt.Sprintf("Remove service %q?\nThis stops and removes its containers. (y/n)", m.service.Name),
				cmds.RequestDockerAction("remove", m.service.Name, false),
			))
		} else if key.Matches(msg, keys.Details.Logs) {
			finalCmds = append(finalCmds, cmds.OpenLogsModal(m.service.Name, false))
		} else if key.Matches(msg, keys.Details.EditService) {
			finalCmds = append(finalCmds, cmds.RequestInlineEdit(m.service.Name))
		} else if key.Matches(msg, keys.Details.EditFile) {
			finalCmds = append(finalCmds, cmds.OpenEditor())
		}
	}

	return m, tea.Batch(finalCmds...)
}

// handleEditKey checks whether a key in edit mode is one of the editor
// control keys (save, open in editor, cancel). It returns the model (which
// may be updated on exit) and the command to run for control keys, or nil
// when the key should be passed through to the textarea as ordinary text.
func (m DetailsPanelModel) handleEditKey(msg tea.KeyPressMsg) (DetailsPanelModel, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Details.Save):
		return m, cmds.RequestSaveService(m.service.Name, []byte(m.editor.Value()))

	case key.Matches(msg, keys.Details.OpenEditor):
		return m, cmds.OpenServiceEditor(m.service.Name)

	case key.Matches(msg, keys.Global.Back):
		if m.hasChanges() {
			return m, cmds.OpenConfirmModal(
				"Discard changes? (y/n)",
				cmds.CancelInlineEdit(),
			)
		}
		return m.exitEditMode()
	}

	return m, nil
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

	basicInfo := BasicInfo(*m.service, bodyWidth)
	runtimeStats := m.renderRuntimeStats(bodyWidth)
	bg := panelBg(m.isFocused)

	var buttons string
	if m.pendingAction != nil {
		buttons = m.renderPendingAction(bodyWidth, bg)
	} else {
		buttons = renderActionButtons(bodyWidth, bg)
	}

	body := lipgloss.JoinVertical(lipgloss.Left, basicInfo, runtimeStats, buttons)
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

// renderRuntimeStats renders a card with live container stats (memory,
// network I/O, disk I/O, uptime) when available. Returns an empty string
// when the service has no running container or no stats data.
func (m DetailsPanelModel) renderRuntimeStats(width int) string {
	if m.service == nil {
		return ""
	}

	container, ok := m.containerForService(m.service.Name)
	if !ok || container.State != "running" {
		return ""
	}

	// Check if we have any stats data at all.
	hasStats := container.MemUsage != "" || container.NetIO != "" || container.BlockIO != ""
	if !hasStats {
		return ""
	}

	wrapper := fitBox(lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(appstyles.Active.Accent).
		Padding(1), width, 0)

	memHeader := lipgloss.NewStyle().Bold(true).Render("Memory: ")
	netHeader := lipgloss.NewStyle().Bold(true).Render("Network: ")
	diskHeader := lipgloss.NewStyle().Bold(true).Render("Disk I/O: ")
	uptimeHeader := lipgloss.NewStyle().Bold(true).Render("Uptime: ")

	var rows []string

	if container.MemUsage != "" {
		memDisplay := formatMemUsage(container.MemUsage, container.MemPerc)
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, memHeader, memDisplay))
	}
	if container.NetIO != "" {
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, netHeader, container.NetIO))
	}
	if container.BlockIO != "" {
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, diskHeader, container.BlockIO))
	}
	if container.RunningFor != "" {
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, uptimeHeader, container.RunningFor))
	}

	if len(rows) == 0 {
		return ""
	}

	info := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return wrapper.Render(info)
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
