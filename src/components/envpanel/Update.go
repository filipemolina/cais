package envpanel

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.loading || len(m.entries) == 0 {
		return m, nil
	}

	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if m.selectedIdx > 0 {
			m.selectedIdx--
			m.revealedIdx = -1 // Re-mask on navigation
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		if m.selectedIdx < len(m.entries)-1 {
			m.selectedIdx++
			m.revealedIdx = -1 // Re-mask on navigation
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
		// Reveal the selected value
		m.Reveal(m.selectedIdx)
		return m, nil
	}

	return m, nil
}
