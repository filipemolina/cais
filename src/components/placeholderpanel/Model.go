package placeholderpanel

import (
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/constants"

	tea "charm.land/bubbletea/v2"
)

// Model is a single full-width panel standing in for a page that has no
// implementation yet. It exists so that navigating to such a page still
// renders a complete frame: AppModel.View drives everything off the pages
// map, and a page with no components used to leave the body empty.
type Model struct {
	title       string
	message     string
	panelWidth  int
	panelHeight int
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m Model) View() tea.View {
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

// New returns a page body that says the page is not built yet.
func New(title string, message string) tea.Model {
	return Model{
		title:   title,
		message: message,
	}
}
