package detailspanel

import (
	"fmt"
	"strings"

	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/chrome"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/cais/src/keys"
	"github.com/filipemolina/cais/src/utils"
	"gopkg.in/yaml.v3"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		if m.service == nil || m.service.Name != service.Name {
			m.urlMessage = ""
			m.applyHint = ""
		}
		// The zero ServiceConfig means "nothing is selected" - configSyncCmds
		// sends it when the services list has just gone empty (e.g. the
		// selected service was deleted) instead of leaving the previous
		// selection standing with nothing behind it. Adopting it as a real
		// service would render its empty name; nil is what every other empty
		// path (View's "Select a service" card) already checks for.
		if service.Name == "" {
			m.service = nil
		} else {
			m.service = &service
		}

	case cmds.AddHealthcheckMsg:
		if msg.Err == nil && m.service != nil && m.service.Name == msg.ServiceName && m.isServiceRunning(msg.ServiceName) {
			m.applyHint = "running: press s to apply - restart won't re-read the compose file"
		}

	case cmds.CycleRestartPolicyMsg:
		if msg.Err == nil && m.service != nil && m.service.Name == msg.ServiceName && m.isServiceRunning(msg.ServiceName) {
			m.applyHint = "running: press s to apply - restart won't re-read the compose file"
		}

	case cmds.InlineEditReadyMsg:
		if msg.Select != nil {
			m.service = msg.Select
		}
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
		// Both body panels are always active now, so the panel answers every
		// key for the selected service (the list owns the keyboard only while
		// a filter is being typed, which this panel never does).
		if m.service == nil {
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
			m.applyHint = ""
			finalCmds = append(finalCmds, cmds.RequestDockerAction(action, m.service.Name, false))
		} else if key.Matches(msg, keys.Details.Remove) {
			finalCmds = append(finalCmds, cmds.OpenConfirmModal(
				fmt.Sprintf("Remove service %q?\nThis stops and removes its containers.", m.service.Name),
				cmds.RequestDockerAction("remove", m.service.Name, false),
			))
		} else if key.Matches(msg, keys.Details.Logs) {
			finalCmds = append(finalCmds, cmds.OpenLogsModal(m.service.Name, false))
		} else if key.Matches(msg, keys.Details.CopyURL) {
			if u, ok := utils.ResolveURL(*m.service, m.host); ok {
				m.urlMessage = "copied " + u.URL
				finalCmds = append(finalCmds, tea.SetClipboard(u.URL))
			}
		} else if key.Matches(msg, keys.Details.Healthcheck) {
			finalCmds = append(finalCmds, cmds.OpenHealthcheckPicker(m.service.Name))
		} else if key.Matches(msg, keys.Details.Boot) {
			m.applyHint = ""
			finalCmds = append(finalCmds, cmds.RequestCycleRestartPolicy(m.service.Name))
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
func (m Model) forwardToEditor(msg tea.Msg) (Model, tea.Cmd) {
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
func (m Model) handleEditKey(msg tea.KeyPressMsg) (Model, tea.Cmd, bool) {
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
func (m Model) currentLine() string {
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
func (m *Model) outdentCurrentLine() {
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
func (m Model) outdentOnBackspace() (Model, bool) {
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
func (m *Model) replaceLine(row int, text string, col int) {
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
func (m Model) hasChanges() bool {
	return string(m.originalFragment) != m.editor.Value()
}

// enterEditMode sets up the textarea with the given fragment and broadcasts
// that the editor now owns the keyboard.
func (m Model) enterEditMode(fragment []byte) (Model, tea.Cmd) {
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
func (m Model) exitEditMode() (Model, tea.Cmd) {
	m.editing = false
	m.originalFragment = nil
	m.saveError = ""
	m.validationError = ""

	return m, cmds.SetEditingState(false)
}

// resizeEditor constrains the textarea to the panel body. It is called on
// layout changes and on entering edit mode.
func (m *Model) resizeEditor() {
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
func (m *Model) updateValidationError() {
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
