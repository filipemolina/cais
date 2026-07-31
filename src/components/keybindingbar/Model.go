package keybindingbar

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/constants"
)

// KeybindingBar is a single-line footer that shows the current page, the
// focused component, and the keys available in that context. It listens for
// SetFocusMsg and SetActivePageMsg to track state — no direct coupling to
// the AppModel.
type Model struct {
	focusedComponent  int
	activePage        string
	terminalWidth     int
	selectedGroup     string
	selectedService   bool
	groupsListEmpty   bool
	servicesListEmpty bool
	composeFile       string
	// composeFileOthers is how many candidates lost to composeFile. The
	// winner is the whole story only when it is the only one, so a +N marks
	// the rest; the help overlay names them.
	composeFileOthers int
	filterState       list.FilterState
	// editing is true while the service details panel is in inline edit
	// mode, so the footer can swap the action keys for the editor keys.
	editing bool
	// pendingAction is true while a docker action is running, so the footer
	// can disable action key hints.
	pendingAction bool
}

func (m Model) Init() tea.Cmd { return nil }

// New builds the footer keybinding bar.
func New() tea.Model {
	return Model{
		focusedComponent:  constants.COMPONENT_BODY_LIST,
		activePage:        "Home",
		groupsListEmpty:   true,
		servicesListEmpty: true,
	}
}
