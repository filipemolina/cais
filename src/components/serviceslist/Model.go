package serviceslist

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/keys"
)

type Model struct {
	list         list.Model
	listDelegate servicesListCustomDelegate
	// activeService is the name of the service the cursor is on. The cursor
	// position IS the selection now (auto-select on navigation), so a stored
	// row number would highlight whichever service moved into that row.
	activeService string
	isFocused     bool
	componentId   int
	fileName      string
	project       *types.Project
	panelWidth    int
	panelHeight   int
	// containers is the latest known container list, used to derive the
	// RUNNING/STOPPED status shown on each service row.
	containers []apptypes.DockerContainer
}

func (m Model) Init() tea.Cmd {
	return nil
}

// OwnsKeyboard reports whether the list is taking every keystroke for itself,
// which it does while a filter is being typed. Same rule as the groups list -
// see groupslist.Model.OwnsKeyboard.
func (m Model) OwnsKeyboard() bool {
	return m.list.FilterState() == list.Filtering
}

// KeepsEsc reports whether the list needs esc for itself. Same rule as the
// groups list - see groupslist.Model.KeepsEsc.
func (m Model) KeepsEsc() bool {
	return m.isFocused && m.list.FilterState() == list.FilterApplied
}

// FilterState exposes how much of the keyboard the list has taken. Same rule
// as the groups list - see groupslist.Model.FilterState.
func (m Model) FilterState() list.FilterState {
	return m.list.FilterState()
}

// New builds the services list.
func New(services []types.ServiceConfig, width int, height int) tea.Model {
	model := Model{
		componentId: 1,
	}

	items := model.buildItems(services)

	// -1 rather than the zero value: no service is active until one is
	// selected, and 0 would render the first row as though one were.
	listDelegate := servicesListCustomDelegate{activeIndex: -1}
	servicesList := list.New(items, listDelegate, width, height)
	servicesList.SetShowHelp(false)
	servicesList.SetShowStatusBar(false)
	// See keys.ListKeyMap: the default map's letter aliases for paging collide
	// with the panel verbs.
	servicesList.KeyMap = keys.ListKeyMap()

	servicesList.Title = "Services"
	servicesList.Paginator.ActiveDot = " ● "
	servicesList.Paginator.InactiveDot = " ○ "
	model.list = servicesList
	model.listDelegate = listDelegate

	return model
}
