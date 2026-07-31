package composefilepanel

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/constants"
	"github.com/filipemolina/stack-stitcher/src/highlight"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

// resizeViewport constrains the inner viewport to the panel box minus the
// frame chrome and the title row.
func (m *Model) resizeViewport() {
	frameW, frameH := chrome.WrapperStyle.GetFrameSize()
	// 2 for the title row and the blank row under it.
	m.viewport.SetWidth(max(1, m.panelWidth-frameW))
	m.viewport.SetHeight(max(1, m.panelHeight-frameH-2))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// Sizing comes from AppModel like every other panel. The single panel
	// takes the whole body row: both panel widths plus the gutter.
	case cmds.SetBodyLayoutMsg:
		m.panelWidth = msg.LeftWidth + constants.BODY_GUTTER_WIDTH + msg.RightWidth
		m.panelHeight = msg.Height
		m.resizeViewport()

	case cmds.SetComposeFileMsg:
		m.filePath = msg.Name

	case cmds.ComposeFileContentsMsg:
		// The path rides with the contents because this is the message the
		// panel is guaranteed to see: SetComposeFileMsg is broadcast while
		// whichever page was active at load time is showing, and only the
		// active page's components receive messages, so a Files page that
		// was inactive at startup never learns the path from it. The
		// contents read fires on page switch, so Name always arrives here.
		m.filePath = msg.Name
		m.readErr = msg.Err
		if msg.Err == nil {
			// content stays raw (the empty-file check reads it); the viewport
			// gets the same bytes with syntax coloring applied. Highlighting
			// is display-only - it changes no byte, so scrolling and the raw
			// view agree. See highlight.YAML.
			m.content = msg.Contents
			m.viewport.SetContent(highlight.YAML(msg.Contents))
		}
		m.loaded = true
		m.viewport.GotoTop()

	case tea.KeyPressMsg:
		if key.Matches(msg, keys.Details.EditFile) {
			return m, cmds.OpenEditor()
		}

		if key.Matches(msg, keys.Files.Browse) {
			return m, cmds.OpenComposeFilePicker()
		}
	}

	// Hand everything else (scroll navigation) to the viewport.
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}
