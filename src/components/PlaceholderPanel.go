package components

import (
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/constants"

	tea "charm.land/bubbletea/v2"
)

// PlaceholderPanelModel is a single full-width panel standing in for a page
// that has no implementation yet. It exists so that navigating to such a page
// still renders a complete frame: AppModel.View drives everything off the
// pages map, and a page with no components used to leave the body empty.
type PlaceholderPanelModel struct {
	title       string
	message     string
	panelWidth  int
	panelHeight int
}

func (m PlaceholderPanelModel) Init() tea.Cmd { return nil }

func (m PlaceholderPanelModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// Sizing comes from AppModel like every other panel. A placeholder is the
	// only component on its page, so it takes the whole body row: both panel
	// widths plus the gutter that would have sat between them.
	case cmds.SetBodyLayoutMsg:
		m.panelWidth = msg.LeftWidth + constants.BODY_GUTTER_WIDTH + msg.RightWidth
		m.panelHeight = msg.Height
	}

	return m, nil
}

func (m PlaceholderPanelModel) View() tea.View {
	bodyWidth := max(1, chrome.PanelBodyWidth(m.panelWidth))
	bodyAvail := max(1, chrome.PanelBodyHeight(m.panelHeight))

	// The panel's title pill already names the page, so the card leads with its
	// state instead of repeating the name.
	//
	// Not focusable, so it always renders on the unfocused panel tier.
	body := chrome.EmptyCard(bodyWidth, bodyAvail, chrome.PanelBg(false),
		"Not built yet", m.message, "", "")

	return tea.NewView(chrome.PanelFrame(m.title, "", false, m.panelWidth, m.panelHeight, body))
}

// PlaceholderPanel returns a page body that says the page is not built yet.
func PlaceholderPanel(title string, message string) tea.Model {
	return PlaceholderPanelModel{
		title:   title,
		message: message,
	}
}
