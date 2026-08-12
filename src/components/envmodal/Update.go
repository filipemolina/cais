package envmodal

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/keys"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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

	if len(m.entries) == 0 {
		// Empty state: only allow 'n' to add the first variable.
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
			return m, func() tea.Msg { return cmds.OpenEnvKeyModalMsg{} }
		case key.Matches(msg, keys.Overlay.Cancel):
			return m, cmds.CloseModal(nil)
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if m.selectedIdx > 0 {
			m.selectedIdx--
			m.revealedIdx = -1
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		if m.selectedIdx < len(m.entries)-1 {
			m.selectedIdx++
			m.revealedIdx = -1
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("home", "g"))):
		m.selectedIdx = 0
		m.revealedIdx = -1
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("end", "G"))):
		m.selectedIdx = len(m.entries) - 1
		m.revealedIdx = -1
		return m, nil

	// Reveal moved off 'v' (now the global opener) to space/enter so the two
	// never collide. Reveal and Copy share the same verb shape as the page.
	case key.Matches(msg, key.NewBinding(key.WithKeys("space", "enter"))):
		if entry := m.selectedVar(); entry != nil {
			m.revealedIdx = m.selectedIdx
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("c"))):
		if entry := m.selectedVar(); entry != nil {
			return m, tea.SetClipboard(entry.Value)
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
		return m, func() tea.Msg { return cmds.OpenEnvKeyModalMsg{} }

	case key.Matches(msg, key.NewBinding(key.WithKeys("e"))):
		if entry := m.selectedVar(); entry != nil {
			return m, func() tea.Msg {
				return cmds.OpenEnvEditModalMsg{Key: entry.Key, Value: entry.Value}
			}
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("d"))):
		if entry := m.selectedVar(); entry != nil {
			return m, func() tea.Msg { return cmds.OpenEnvDeleteConfirmMsg{Key: entry.Key} }
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("o"))):
		return m, func() tea.Msg { return cmds.OpenEnvRawEditMsg{} }

	case key.Matches(msg, key.NewBinding(key.WithKeys("E"))):
		// Open the .env file in $EDITOR. The modal closes so the editor
		// takes the terminal, like the Files page does.
		return m, cmds.CloseModal(cmds.OpenEditor())

	case key.Matches(msg, keys.Overlay.Cancel):
		return m, cmds.CloseModal(nil)
	}

	return m, nil
}
