package healthcheckpickermodal

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(keyMsg, keys.Overlay.Cancel):
			return m, cmds.CloseModal(nil)

		case key.Matches(keyMsg, keys.Overlay.Submit):
			return m.submit()

		// Cursor movement bypasses list.Update's own key handling
		// entirely - the bubbles default keymap claims h/l/b/u/f/d/g/G/q
		// among others, every one of which the port field needs to accept
		// as ordinary typing while the generic template is highlighted.
		case keyMsg.Code == tea.KeyUp:
			m.list.CursorUp()
			m.errMsg = ""
			return m, nil

		case keyMsg.Code == tea.KeyDown:
			m.list.CursorDown()
			m.errMsg = ""
			return m, nil
		}

		// Visibility is derived from the selection (docs/plans/healthcheck-
		// insertion.md's "UX: the modal"): while the generic template is
		// highlighted, every other key types into the port field rather
		// than being a list command the field has no use for.
		if t, ok := m.selectedTemplate(); ok && t.Generic {
			var cmd tea.Cmd
			m.portInput, cmd = m.portInput.Update(keyMsg)
			return m, cmd
		}

		return m, nil
	}

	// Non-key messages (cursor blink ticks, window size) still need to
	// reach both sub-components.
	var listCmd, inputCmd tea.Cmd
	m.list, listCmd = m.list.Update(msg)
	m.portInput, inputCmd = m.portInput.Update(msg)
	return m, tea.Batch(listCmd, inputCmd)
}

func (m Model) submit() (tea.Model, tea.Cmd) {
	t, ok := m.selectedTemplate()
	if !ok {
		return m, nil
	}

	var port string
	if t.Generic {
		port = strings.TrimSpace(m.portInput.Value())
		if port == "" {
			m.errMsg = "Port can't be empty"
			return m, nil
		}
		if n, err := strconv.ParseUint(port, 10, 16); err != nil || n == 0 {
			m.errMsg = fmt.Sprintf("%q is not a valid port", port)
			return m, nil
		}
	}

	return m, cmds.CloseModal(cmds.RequestAddHealthcheck(m.serviceName, t, port))
}
