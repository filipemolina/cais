package components

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/constants"
	"github.com/filipemolina/stack-stitcher/src/highlight"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

// ComposeFilePanel replaces the PlaceholderPanel on the Files page. The
// minimal version shows the active compose file's path in the title and a
// read-only, scrollable view of its raw contents. E opens the file in the
// user's editor.
//
// The panel is the sole component on its page, so it fills the whole body
// row (both panel widths plus the gutter). It is not split into a list and
// a details pane - that is a later-phase extension to browse multiple
// compose files.
// The panel is the only component on its page, so it is always focused:
// there is no second panel for Tab to move to, and tracking SetFocusMsg
// would let Tab strand focus on a component id that does not exist here,
// blurring the one panel the page has. Always-focused also means the E key
// and scrolling always work, matching what the footer advertises.
type ComposeFilePanelModel struct {
	viewport    viewport.Model
	filePath    string
	content     string
	readErr     error
	loaded      bool
	panelWidth  int
	panelHeight int
}

func (m ComposeFilePanelModel) Init() tea.Cmd { return nil }

func (m ComposeFilePanelModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	}

	// Hand everything else (scroll navigation) to the viewport.
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// resizeViewport constrains the inner viewport to the panel box minus the
// frame chrome and the title row.
func (m *ComposeFilePanelModel) resizeViewport() {
	frameW, frameH := wrapperStyle.GetFrameSize()
	// 2 for the title row and the blank row under it.
	m.viewport.SetWidth(max(1, m.panelWidth-frameW))
	m.viewport.SetHeight(max(1, m.panelHeight-frameH-2))
}

func (m ComposeFilePanelModel) View() tea.View {
	// Always the focused tier: see the note on the model.
	bg := panelBg(true)

	bodyWidth := max(1, panelBodyWidth(m.panelWidth))
	bodyAvail := max(1, panelBodyHeight(m.panelHeight))

	var body string
	switch {
	case m.filePath == "":
		body = renderEmptyCard(bodyWidth, bodyAvail, bg,
			"No compose file",
			"No compose file is loaded. Run Stack Stitcher from a directory with a compose file, or use --file/--dir.",
			"", "")
	case m.readErr != nil:
		body = renderEmptyCard(bodyWidth, bodyAvail, bg,
			"Could not read file",
			m.readErr.Error(),
			"E", "edit in $EDITOR")
	case m.loaded && strings.TrimSpace(m.content) == "":
		body = renderEmptyCard(bodyWidth, bodyAvail, bg,
			"Empty file",
			"The compose file is empty.",
			"E", "edit in $EDITOR")
	default:
		vp := m.viewport.View()
		// Seal the viewport against the panel background. JoinVertical below
		// would pad the shorter rows out to the panel width with unstyled
		// spaces, so the viewport output must already carry the background.
		vp = appstyles.FillBackground(bg, vp)
		body = lipgloss.NewStyle().
			Width(bodyWidth).
			Height(bodyAvail).
			MaxHeight(bodyAvail).
			Background(bg).
			Render(vp)
	}

	// The title shows the file path when one is loaded. The path is the
	// answer to "which file am I looking at?" and earns its place on the
	// title row the same way a status pill does on the details panels.
	title := "Files"
	titleRight := ""
	if m.filePath != "" {
		titleRight = lipgloss.NewStyle().
			Foreground(appstyles.Active.TextDim).
			Render(m.filePath)
	}

	screen := renderPanelFrame(title, titleRight, true, m.panelWidth, m.panelHeight, body)
	return tea.NewView(screen)
}

// errorContent renders a read failure as plain text the viewport can
// display when the file could not be loaded.
func errorContent(message string) string {
	return lipgloss.NewStyle().
		Foreground(appstyles.Active.Danger).
		Render("Error: " + message)
}

// composeFileViewportKeyMap returns a viewport keymap with the letter keys
// stripped. The viewport's DefaultKeyMap claims f, b, u, d, h, l, k, j -
// the same collisions that forced the list keymap work. A read-only file
// viewer has no use for horizontal scrolling or vim-style half-pages, so
// the letter keys are unbound and only the arrows and pgup/pgdn remain.
func composeFileViewportKeyMap() viewport.KeyMap {
	unbound := key.NewBinding()

	return viewport.KeyMap{
		PageDown:     key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdn", "page down")),
		PageUp:       key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "page up")),
		HalfPageUp:   key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "½ page up")),
		HalfPageDown: key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "½ page down")),
		Up:           key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:         key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Left:         unbound,
		Right:        unbound,
	}
}

// ComposeFilePanel builds the read-only file viewer for the Files page.
func ComposeFilePanel() tea.Model {
	vp := viewport.New()
	vp.KeyMap = composeFileViewportKeyMap()

	return ComposeFilePanelModel{
		viewport: vp,
	}
}
