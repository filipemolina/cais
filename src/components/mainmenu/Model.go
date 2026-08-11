package mainmenu

import (
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/apptypes"
)

// Model is the top nav bar. It is not focusable and handles no keys:
// pages are switched with the global digit keys that it advertises by
// rendering each tab's digit before its label. All it tracks is which page is
// active, so it can highlight that tab.
type Model struct {
	items             []string
	selectedItemIndex int
	terminalWidth     int
}

func (m Model) Init() tea.Cmd {
	return nil
}

// New builds the top nav bar.
func New() tea.Model {
	items := []string{}

	for _, page := range apptypes.PageTitles {
		items = append(items, page)
	}

	m := Model{items: items}

	return m
}
