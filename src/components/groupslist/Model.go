package groupslist

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

type Model struct {
	list         list.Model
	listDelegate GroupsListCustomDelegate
	// activeGroup is the name of the group the cursor is on. The cursor
	// position IS the selection now (auto-select on navigation), so a stored
	// row number would highlight whichever group moved into that row.
	activeGroup string
	isFocused   bool
	componentId int
	stats       cmds.SetHomeStatsMsg
	hasStats    bool
	panelWidth  int
	panelHeight int
}

func (m Model) Init() tea.Cmd {
	return nil
}

// OwnsKeyboard reports whether the list is taking every keystroke for itself,
// which it does while the user is typing a filter: n, d and q are letters then,
// not commands. Only while typing - once a filter is applied and the cursor is
// back in the rows, the panel keys mean what they always mean, and esc clears
// the filter. See model.AppModel.keyboardOwned.
func (m Model) OwnsKeyboard() bool {
	return m.list.FilterState() == list.Filtering
}

// KeepsEsc reports whether the list needs esc for itself: an applied filter
// is cleared by esc alone, and the key only reaches the list while the list
// is focused. AppModel's "back" checks this before it takes focus away - see
// model.AppModel.escKept.
func (m Model) KeepsEsc() bool {
	return m.isFocused && m.list.FilterState() == list.FilterApplied
}

// FilterState exposes how much of the keyboard the list has taken, so
// AppModel can snapshot it into the help overlay's context.
func (m Model) FilterState() list.FilterState {
	return m.list.FilterState()
}

// footerHeight is the rows the stats line takes below the list.
func (m Model) footerHeight() int {
	if !m.hasStats {
		return 0
	}

	return 1
}

// New builds the groups list.
func New(groups []string, width int, height int) tea.Model {
	var items []list.Item

	for _, group := range groups {
		items = append(items, apptypes.GroupListItem(group))
	}

	// -1 rather than the zero value: no group is active until one is
	// selected, and 0 would render the first row as though one were.
	listDelegate := GroupsListCustomDelegate{activeIndex: -1}
	servicesList := list.New(items, listDelegate, width, height)
	servicesList.SetShowHelp(false)
	servicesList.SetShowStatusBar(false)
	// Without this the list keeps list.DefaultKeyMap, which claims d, f, l, h,
	// b, u, q, esc and ? - keys this app spends elsewhere. See keys.ListKeyMap.
	servicesList.KeyMap = keys.ListKeyMap()

	servicesList.Title = "Groups"
	servicesList.Paginator.ActiveDot = " ● "
	servicesList.Paginator.InactiveDot = " ○ "

	model := Model{
		list:         servicesList,
		listDelegate: listDelegate,
		componentId:  1,
	}

	return model
}
