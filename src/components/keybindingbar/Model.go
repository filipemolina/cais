package keybindingbar

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/apptypes"
)

// KeybindingBar is a single-line footer that shows the current page and the
// keys available in that context. Both body panels are always active now, so
// it no longer tracks a focused component — it listens for SetActivePageMsg to
// track state, with no direct coupling to the AppModel.
type Model struct {
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
	// ungroupedMaterialized is true when the reserved ungrouped row is backed
	// by a written profile tag rather than derived, so the footer can
	// advertise the row's 'A' verb (adopt vs release).
	ungroupedMaterialized bool
	// backupsListEmpty is whether the backup store has any stored versions,
	// so the bar can drop the filter key on an empty one. It comes straight
	// off cmds.BackupListMsg: the bar sees every message, and AppModel does
	// not track the store's contents.
	backupsListEmpty bool
	// backupsFocus is which half of the Backups page the arrows are driving,
	// so the bar says "navigate" over the list and "scroll" over the preview.
	// It is the only page with focus left; see apptypes.BackupsFocus.
	backupsFocus apptypes.BackupsFocus
}

func (m Model) Init() tea.Cmd { return nil }

// New builds the footer keybinding bar.
func New() tea.Model {
	return Model{
		activePage:        "Home",
		groupsListEmpty:   true,
		servicesListEmpty: true,
		backupsListEmpty:  true,
	}
}
