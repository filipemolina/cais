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

	style := lipgloss.NewStyle().Foreground(appstyles.SecondaryFontColor)
	if index == m.Index() {
		style = style.Foreground(appstyles.PrimaryFontColor).Bold(true)
	}

	fmt.Fprint(w, style.Render(item.Title()))
}

type ServiceChecklistModalModel struct {
	groupName string
	list      list.Model
}

func (m ServiceChecklistModalModel) Init() tea.Cmd {
	return nil
}

func (m ServiceChecklistModalModel) checkedServiceNames() []string {
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
			if checked := m.checkedServiceNames(); len(checked) > 0 {
				return m, cmds.CloseModal(cmds.CreateGroup(m.groupName, checked))
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
// TextMuted, not the bar's TextDim: this sits on the modal's light surface,
// where TextDim barely separates from the background.
func checklistHints() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		renderKeyHints([]KeyHint{
			hintFor(keys.List.Navigate),
			hintFor(keys.Overlay.Toggle),
		}, appstyles.TextMuted),
		renderKeyHints([]KeyHint{
			// Enter is "confirm" everywhere; here what it confirms is worth
			// naming, since it is the step that writes to the compose file.
			hintAs(keys.Overlay.Submit, "create group"),
			hintFor(keys.Overlay.Cancel),
		}, appstyles.TextMuted),
	)
}

func (m ServiceChecklistModalModel) View() tea.View {
	content := lipgloss.JoinVertical(lipgloss.Left, m.list.View(), "", checklistHints())

	return tea.NewView(modalSurface(appstyles.PanelBackgroundColor, content))
}

// ServiceChecklistModal is step 2 of the create-group flow: pick which
// services get tagged with groupName. Space toggles the highlighted
// service, Enter confirms (requires at least one checked), Esc cancels the
// whole create flow.
func ServiceChecklistModal(groupName string, serviceNames []string) tea.Model {
	items := make([]list.Item, 0, len(serviceNames))
	for _, name := range serviceNames {
		items = append(items, apptypes.CheckableServiceItem{Name: name})
	}

	// +2 for the title row and the blank row under it, which is what the list
	// leaves for its items. Without the pagination row switched off, the list
	// spends a row on a paginator that this modal has no use for - it is sized
	// to show every service at once - and takes that row out of the items,
	// silently pushing the last services onto an unreachable second page.
	checklist := list.New(items, serviceChecklistDelegate{}, 40, len(items)+2)
	checklist.Title = fmt.Sprintf("Select services for %q", groupName)
	checklist.SetShowHelp(false)
	checklist.SetShowStatusBar(false)
	checklist.SetShowPagination(false)
	checklist.Styles.Title = checklist.Styles.Title.Background(appstyles.PrimaryColor)

	return ServiceChecklistModalModel{
		groupName: groupName,
		list:      checklist,
	}
}
