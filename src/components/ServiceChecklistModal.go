package components

import (
	"fmt"
	"io"
	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/apptypes"
	"stack-stitcher/src/cmds"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type serviceChecklistDelegate struct{}

func (d serviceChecklistDelegate) Height() int                            { return 1 }
func (d serviceChecklistDelegate) Spacing() int                           { return 0 }
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
	profileName string
	list        list.Model
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
		switch keyMsg.String() {
		case "esc":
			return m, cmds.CloseModal(nil)

		case "space":
			index := m.list.GlobalIndex()
			if item, ok := m.list.SelectedItem().(apptypes.CheckableServiceItem); ok {
				item.Checked = !item.Checked
				finalCmds = append(finalCmds, m.list.SetItem(index, item))
			}

		case "enter":
			if checked := m.checkedServiceNames(); len(checked) > 0 {
				return m, cmds.CloseModal(cmds.CreateProfile(m.profileName, checked))
			}
		}
	}

	var listCmd tea.Cmd
	m.list, listCmd = m.list.Update(msg)
	finalCmds = append(finalCmds, listCmd)

	return m, tea.Batch(finalCmds...)
}

func (m ServiceChecklistModalModel) View() tea.View {
	style := lipgloss.NewStyle().
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(appstyles.PrimaryColor).
		Background(appstyles.PanelBackgroundColor)

	return tea.NewView(style.Render(m.list.View()))
}

// ServiceChecklistModal is step 2 of the create-profile flow: pick which
// services get tagged with profileName. Space toggles the highlighted
// service, Enter confirms (requires at least one checked), Esc cancels the
// whole create flow.
func ServiceChecklistModal(profileName string, serviceNames []string) tea.Model {
	items := make([]list.Item, 0, len(serviceNames))
	for _, name := range serviceNames {
		items = append(items, apptypes.CheckableServiceItem{Name: name})
	}

	checklist := list.New(items, serviceChecklistDelegate{}, 40, len(items)+2)
	checklist.Title = fmt.Sprintf("Select services for %q", profileName)
	checklist.SetShowHelp(false)
	checklist.SetShowStatusBar(false)
	checklist.Styles.Title = checklist.Styles.Title.Background(appstyles.PrimaryColor)

	return ServiceChecklistModalModel{
		profileName: profileName,
		list:        checklist,
	}
}
