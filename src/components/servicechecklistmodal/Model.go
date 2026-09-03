package servicechecklistmodal

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/keys"
)

type Model struct {
	groupName string
	list      list.Model
	// isEdit selects which request the modal emits on Enter: a create for
	// a new group, or an edit reconciliation for an existing one.
	isEdit bool
}

func (m Model) Init() tea.Cmd {
	return nil
}

// CheckedNames returns the names of the currently checked services, in
// list order. Exported so the model tests can assert the pre-checked state
// an edit modal opened with.
func (m Model) CheckedNames() []string {
	var names []string

	for _, listItem := range m.list.Items() {
		if item, ok := listItem.(apptypes.CheckableServiceItem); ok && item.Checked {
			names = append(names, item.Name)
		}
	}

	return names
}

// checklist builds the inner list shared by both constructors. termHeight is
// the terminal height in rows; a compose file can define more services than
// the screen has room for, so the list is sized to fit. See chrome.ModalListHeight.
func checklist(serviceNames []string, preselected map[string]bool, termHeight int) list.Model {
	items := make([]list.Item, 0, len(serviceNames))
	for _, name := range serviceNames {
		items = append(items, apptypes.CheckableServiceItem{
			Name:    name,
			Checked: preselected[name],
		})
	}

	// The title is rendered by chrome.ModalTitle in the View function, not by the
	// list itself. Pagination is off while every service fits, because the
	// paginator would take a row out of the items and silently push the last
	// services onto an unreachable second page. Once the terminal is too
	// short to show them all that is no longer a lie the modal can tell, so
	// the paginator comes back and says which page the cursor is on.
	visible := chrome.ModalListHeight(len(items), termHeight)

	cl := list.New(items, serviceChecklistDelegate{}, 40, visible)
	cl.KeyMap = keys.ModalListKeyMap()
	cl.SetShowTitle(false)
	cl.SetShowHelp(false)
	cl.SetShowStatusBar(false)
	cl.SetShowPagination(visible < len(items))

	return cl
}

// New is step 2 of the create-group flow: pick which services get tagged
// with groupName. Space toggles the highlighted service, Enter confirms
// (requires at least one checked), Esc cancels the whole create flow.
func New(groupName string, serviceNames []string, termHeight int) tea.Model {
	cl := checklist(serviceNames, nil, termHeight)

	return Model{
		groupName: groupName,
		list:      cl,
	}
}

// NewForEdit reopens the service checklist to edit an existing group's
// membership. Services that already belong to the group are pre-checked;
// Enter saves the diff (including empty, which removes the group entirely).
// Esc cancels without writing.
func NewForEdit(groupName string, serviceNames []string, currentMembers []string, termHeight int) tea.Model {
	preselected := make(map[string]bool, len(currentMembers))
	for _, name := range currentMembers {
		preselected[name] = true
	}

	cl := checklist(serviceNames, preselected, termHeight)

	// Move the cursor to the first unchecked service so the user lands
	// on something actionable rather than having to arrow past members
	// they have already confirmed. If every service is checked, the cursor
	// stays at the top.
	for i, item := range cl.Items() {
		if checked, ok := item.(apptypes.CheckableServiceItem); ok && !checked.Checked {
			cl.Select(i)
			break
		}
	}

	return Model{
		groupName: groupName,
		list:      cl,
		isEdit:    true,
	}
}
