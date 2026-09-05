package envmodal

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/keys"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// The terminal can change size under an open modal; see
		// chrome.ResizeModalList.
		m.termHeight = msg.Height
		resizeEnvList(&m.list, msg.Height)
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case cmds.EnvFileContentsMsg:
		// A parse error is non-fatal: the table still shows what parsed.
		if msg.Err != nil {
			m.SetLoadError(msg.Err)
			return m, nil
		}
		m.SetEntries(msg.Path, msg.Entries, 0) // TODO: count parse errors
		return m, nil

	case cmds.SaveEnvFileMsg:
		// The save is handled by AppModel (reload + reload). Re-request the
		// contents so the table reflects the write once the modal is back.
		if m.envPath != "" {
			return m, cmds.GetEnvFileContents(m.envPath)
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}

	if len(m.list.Items()) == 0 {
		// Empty state: only allow 'n' to add the first variable.
		switch {
		case key.Matches(msg, keys.List.New):
			return m, func() tea.Msg { return cmds.OpenEnvKeyModalMsg{} }
		case key.Matches(msg, keys.Overlay.Cancel):
			return m, cmds.CloseModal(nil)
		}
		return m, nil
	}

	// The modal's own verbs are matched before the list sees the keystroke,
	// so a verb the list also claims resolves here. Everything unmatched
	// falls through to the list, which owns cursor movement.
	switch {
	case key.Matches(msg, keys.Env.Reveal):
		if entry := m.selectedVar(); entry != nil {
			m.revealed = m.list.Index()
		}
		return m, nil

	case key.Matches(msg, keys.Env.Copy):
		if entry := m.selectedVar(); entry != nil {
			return m, tea.SetClipboard(entry.Value)
		}
		return m, nil

	case key.Matches(msg, keys.Env.RawEdit):
		return m, func() tea.Msg { return cmds.OpenEnvRawEditMsg{} }

	case key.Matches(msg, keys.List.New):
		return m, func() tea.Msg { return cmds.OpenEnvKeyModalMsg{} }

	case key.Matches(msg, keys.List.Edit):
		if entry := m.selectedVar(); entry != nil {
			return m, func() tea.Msg {
				return cmds.OpenEnvEditModalMsg{Key: entry.Key, Value: entry.Value}
			}
		}
		return m, nil

	case key.Matches(msg, keys.List.Delete):
		if entry := m.selectedVar(); entry != nil {
			return m, func() tea.Msg { return cmds.OpenEnvDeleteConfirmMsg{Key: entry.Key} }
		}
		return m, nil

	case key.Matches(msg, keys.Details.EditFile):
		// Open the .env file in $EDITOR. The modal closes so the editor
		// takes the terminal, like the Files page does.
		return m, cmds.CloseModal(cmds.OpenEditor())

	case key.Matches(msg, keys.Overlay.Cancel):
		return m, cmds.CloseModal(nil)
	}

	before := m.list.Index()

	var listCmd tea.Cmd
	m.list, listCmd = m.list.Update(msg)

	// Moving off a revealed row re-hides it. A value is revealed for as long
	// as you are looking at it and no longer - leaving it revealed behind the
	// cursor is how a secret ends up on someone else's screen.
	if m.list.Index() != before {
		m.revealed = -1
	}

	return m, listCmd
}
