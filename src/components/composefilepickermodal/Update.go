package composefilepickermodal

import (
	"path/filepath"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/keys"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var finalCmds []tea.Cmd

	// The terminal can change size while the modal is open. Re-fit rather
	// than keep the height chosen at construction, or the bottom border ends
	// up off screen. See chrome.ResizeModalList.
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		chrome.ResizeModalList(&m.list, len(m.list.Items()), size.Height)
		return m, nil
	}

	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(keyMsg, keys.Overlay.Cancel):
			return m, cmds.CloseModal(nil)

		case key.Matches(keyMsg, keys.Overlay.Submit):
			if item, ok := m.list.SelectedItem().(apptypes.ComposeFileItem); ok {
				return m, cmds.CloseModal(cmds.SwitchComposeFile(filepath.Join(m.dir, item.Name)))
			}
		}
	}

	var listCmd tea.Cmd
	m.list, listCmd = m.list.Update(msg)
	finalCmds = append(finalCmds, listCmd)

	return m, tea.Batch(finalCmds...)
}
