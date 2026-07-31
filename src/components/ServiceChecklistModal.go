package components

import (
	"fmt"
	"io"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

type serviceChecklistDelegate struct{}

func (d serviceChecklistDelegate) Height() int                             { return 1 }
func (d serviceChecklistDelegate) Spacing() int                            { return 0 }
func (d serviceChecklistDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d serviceChecklistDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(apptypes.CheckableServiceItem)
	if !ok {
		return
	}

	style := lipgloss.NewStyle().Foreground(appstyles.Active.TextMuted)
	if index == m.Index() {
		style = style.Foreground(appstyles.Active.TextPrimary).Bold(true)
	}

	fmt.Fprint(w, style.Render(item.Title()))
}

type ServiceChecklistModalModel struct {
	groupName string
	list      list.Model
	// isEdit selects which request the modal emits on Enter: a create for
	// a new group, or an edit reconciliation for an existing one.
	isEdit bool
}

func (m ServiceChecklistModalModel) Init() tea.Cmd {
	return nil
}

// CheckedNames returns the names of the currently checked services, in
// list order. Exported so the model tests can assert the pre-checked state
// an edit modal opened with.
func (m ServiceChecklistModalModel) CheckedNames() []string {
	var names []string

	for _, listItem := range m.list.Items() {
		if item, ok := listItem.(apptypes.CheckableServiceItem); ok && item.Checked {
			names = append(names, item.Name)
		}
	}

	return names
}

func (m ServiceChecklistModalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var finalCmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(keyMsg, keys.Overlay.Cancel):
			return m, cmds.CloseModal(nil)

		case key.Matches(keyMsg, keys.Overlay.Toggle):
			index := m.list.GlobalIndex()
			if item, ok := m.list.SelectedItem().(apptypes.CheckableServiceItem); ok {
				item.Checked = !item.Checked
				finalCmds = append(finalCmds, m.list.SetItem(index, item))
			}

		case key.Matches(keyMsg, keys.Overlay.Submit):
			checked := m.CheckedNames()
			if m.isEdit {
				// Editing an existing group allows empty membership:
				// unchecking every service removes the group from the
				// list, which is the same outcome as deleting it.
				return m, cmds.CloseModal(cmds.RequestEditGroup(m.groupName, checked))
			}
			if len(checked) > 0 {
				return m, cmds.CloseModal(cmds.RequestCreateGroup(m.groupName, checked))
			}
		}
	}

	var listCmd tea.Cmd
	m.list, listCmd = m.list.Update(msg)
	finalCmds = append(finalCmds, listCmd)

	return m, tea.Batch(finalCmds...)
}

// checklistHints is the modal's own help line. The footer bar is hidden
// behind the modal while this is open, so the keys it takes over - space,
// enter, esc - have to be advertised here or nowhere. Two lines rather than
// one so the modal stays as narrow as its list.
//
// submitDesc names what Enter confirms in this mode: "create group" for a
// new group, "save changes" for an edit. Enter is "confirm" everywhere;
// here what it confirms is worth naming, since it is the step that writes
// to the compose file.
//
// TextMuted, not the bar's TextDim: this sits on the modal's light surface,
// where TextDim barely separates from the background.
func checklistHints(submitDesc string) string {
	return lipgloss.JoinVertical(lipgloss.Left,
		renderKeyHints([]KeyHint{
			hintFor(keys.List.Navigate),
			hintFor(keys.Overlay.Toggle),
		}, appstyles.Active.TextMuted),
		renderKeyHints([]KeyHint{
			hintAs(keys.Overlay.Submit, submitDesc),
			hintFor(keys.Overlay.Cancel),
		}, appstyles.Active.TextMuted),
	)
}

func (m ServiceChecklistModalModel) View() tea.View {
	submitDesc := "create group"
	if m.isEdit {
		submitDesc = "save changes"
	}

	title := fmt.Sprintf("Select services for %q", m.groupName)
	if m.isEdit {
		title = fmt.Sprintf("Edit members of %q", m.groupName)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, modalTitle(title), m.list.View(), "", checklistHints(submitDesc))

	return tea.NewView(modalSurface(appstyles.Active.ModalBg, content))
}

// checklist builds the inner list shared by both constructors. termHeight is
// the terminal height in rows; a compose file can define more services than
// the screen has room for, so the list is sized to fit. See modalListHeight.
func checklist(serviceNames []string, preselected map[string]bool, termHeight int) list.Model {
	items := make([]list.Item, 0, len(serviceNames))
	for _, name := range serviceNames {
		items = append(items, apptypes.CheckableServiceItem{
			Name:    name,
			Checked: preselected[name],
		})
	}

	// The title is rendered by modalTitle in the View function, not by the
	// list itself. Pagination is off while every service fits, because the
	// paginator would take a row out of the items and silently push the last
	// services onto an unreachable second page. Once the terminal is too
	// short to show them all that is no longer a lie the modal can tell, so
	// the paginator comes back and says which page the cursor is on.
	visible := modalListHeight(len(items), termHeight)

	cl := list.New(items, serviceChecklistDelegate{}, 40, visible)
	cl.SetShowTitle(false)
	cl.SetShowHelp(false)
	cl.SetShowStatusBar(false)
	cl.SetShowPagination(visible < len(items))

	return cl
}

// ServiceChecklistModal is step 2 of the create-group flow: pick which
// services get tagged with groupName. Space toggles the highlighted
// service, Enter confirms (requires at least one checked), Esc cancels the
// whole create flow.
func ServiceChecklistModal(groupName string, serviceNames []string, termHeight int) tea.Model {
	cl := checklist(serviceNames, nil, termHeight)

	return ServiceChecklistModalModel{
		groupName: groupName,
		list:      cl,
	}
}

// ServiceChecklistModalForEdit reopens the service checklist to edit an
// existing group's membership. Services that already belong to the group
// are pre-checked; Enter saves the diff (including empty, which removes
// the group entirely). Esc cancels without writing.
func ServiceChecklistModalForEdit(groupName string, serviceNames []string, currentMembers []string, termHeight int) tea.Model {
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

	return ServiceChecklistModalModel{
		groupName: groupName,
		list:      cl,
		isEdit:    true,
	}
}
