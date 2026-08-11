package envpanel

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/constants"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// Sizing comes from AppModel like every other panel; deriving it from
	// WindowSizeMsg here would leave the panel at width 0 whenever Env wasn't
	// the active page at resize time. As the sole component on its page, it
	// takes the whole body row: both panel widths plus the gutter.
	case cmds.SetBodyLayoutMsg:
		m.SetSize(msg.LeftWidth+constants.BODY_GUTTER_WIDTH+msg.RightWidth, msg.Height)
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	case cmds.EnvFileContentsMsg:
		m.SetEnvEntries(msg.Path, msg.Entries, 0) // TODO: count parse errors
		if msg.Err != nil {
			m.SetLoadError(msg.Err)
		}
		return m, nil
	case cmds.SaveEnvFileMsg:
		// The save message is handled by the main model, which triggers a reload.
		// Just update loading state here if needed.
		return m, nil
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}

	if len(m.entries) == 0 {
		// Empty state: only allow 'n' to add first variable
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
			return m, func() tea.Msg { return cmds.OpenEnvKeyModalMsg{} }
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

	case key.Matches(msg, key.NewBinding(key.WithKeys("v", "enter"))):
		if entry := m.getSelectedVar(); entry != nil {
			m.Reveal(m.selectedIdx)
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("c"))):
		if entry := m.getSelectedVar(); entry != nil {
			return m, tea.SetClipboard(entry.Value)
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
		return m, func() tea.Msg { return cmds.OpenEnvKeyModalMsg{} }

	case key.Matches(msg, key.NewBinding(key.WithKeys("e"))):
		if entry := m.getSelectedVar(); entry != nil {
			return m, func() tea.Msg {
				return cmds.OpenEnvEditModalMsg{Key: entry.Key, Value: entry.Value}
			}
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("d"))):
		if entry := m.getSelectedVar(); entry != nil {
			return m, func() tea.Msg { return cmds.OpenEnvDeleteConfirmMsg{Key: entry.Key} }
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("o"))):
		return m, func() tea.Msg { return cmds.OpenEnvRawEditMsg{} }

	case key.Matches(msg, key.NewBinding(key.WithKeys("E"))):
		// Open the .env file in $EDITOR
		return m, func() tea.Msg { return cmds.OpenEditorMsg{} }
	}

	return m, nil
}

func (m *Model) getSelectedVar() *cmds.EnvEntry {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.entries) {
		return nil
	}
	entry := &m.entries[m.selectedIdx]
	if entry.Source != "var" {
		return nil
	}
	return entry
}
