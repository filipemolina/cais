package helpoverlay

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/keys"
)

// helpOverlayMaxWidth caps the content column so a long description wraps to
// a couple of lines on wide terminals rather than stretching into one
// unreadable line. Each entry is already one key/description row, so this
// only wraps within a single row, never across rows.
const helpOverlayMaxWidth = 64

// Model is the ? overlay: every key in the app, grouped by scope
// and rendered from keys.Catalog, so what it says is what the handlers do.
// Rows the user could not press in the screen it was opened from are dimmed.
// It also names the compose-file candidates that lost to the loaded one - the
// footer only has room to count them.
//
// The catalog is more than one terminal's worth of lines once every entry
// gets its own row, so the content is windowed and scrolls with ↑/↓
// (keys.Overlay.Navigation), and narrowable with a `/`-fuzzy filter over
// each entry's key and description.
type Model struct {
	catalog      []keys.Scope
	composeFiles []string
	termWidth    int
	termHeight   int
	// offset is the first visible content line. Content is one flat list of
	// already-wrapped lines by the time it is windowed, so scrolling is per
	// line rather than per scope: a scope taller than the window is still
	// readable.
	offset int

	// filterInput is the `/`-filter's text box. filterTyping is true while it
	// owns the keyboard (every keystroke but enter/esc lands in it);
	// filterApplied is true once enter locks the query in and the input
	// blurs, matching keys.Overlay.Navigation back to scrolling. filterQuery
	// mirrors filterInput's value live, so the catalog narrows as the user
	// types rather than waiting for enter.
	filterInput   textinput.Model
	filterTyping  bool
	filterApplied bool
	filterQuery   string
}

func (m Model) Init() tea.Cmd { return nil }

// New builds the help overlay for the screen described by ctx (which
// keys are pressable), the compose-file candidates in priority order, and the
// terminal size for wrapping and windowing.
func New(ctx keys.Context, composeFiles []string, termWidth, termHeight int) tea.Model {
	fi := textinput.New()
	// The bubbles default prompt is a hardcoded ANSI-white "> ", which would
	// render between the bar's "/" and the query.
	fi.Prompt = ""
	return Model{
		catalog:      keys.Catalog(ctx),
		composeFiles: composeFiles,
		termWidth:    termWidth,
		termHeight:   termHeight,
		filterInput:  fi,
	}
}
