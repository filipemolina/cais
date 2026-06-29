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

/*
 * Styling by creating a custom delegate
 */

type customDelegate struct{}

func (d customDelegate) Height() int                             { return 6 }
func (d customDelegate) Spacing() int                            { return 0 }
func (d customDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

// Render handles the actual drawing of the item
func (d customDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	// Cast the generic list.Item back to our specific myItem
	item, ok := listItem.(apptypes.ContainerListItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()
	isActive := false

	wrapperStyle := lipgloss.NewStyle().
		MarginBottom(1).
		Width(m.Width()).
		Padding(1)

	titleStyle := lipgloss.NewStyle().Bold(true).Width(m.Width())
	descriptionStyle := lipgloss.NewStyle().Foreground(appstyles.SecondaryFontColor)

	if isSelected {
		wrapperStyle = wrapperStyle.
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderLeftForeground(appstyles.PrimaryColor).
			Background(lipgloss.Color("#3F3F3F"))

	} else if isActive {
		// Highlight text if active, but not currently selected
		wrapperStyle = wrapperStyle.
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderLeftForeground(appstyles.PrimaryFontColor)

	} else {
		// Default unselected, inactive state
		wrapperStyle = wrapperStyle.
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderLeftForeground(appstyles.SecondaryFontColor)

	}

	title := titleStyle.Render(item.Title())
	description := descriptionStyle.Render(item.Description())

	// Print the styled string to the Bubble Tea io.Writer
	fmt.Fprint(w, wrapperStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, "", description)))
}

/*
 * Implementation of tea.Model
 */

type ServicesListModel struct {
	list list.Model
}

func (m ServicesListModel) Init() tea.Cmd {
	return nil
}

func (m ServicesListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var finalCmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appstyles.DocStyle.GetFrameSize()

		m.list.SetSize(
			((msg.Width-h)*4/10)-1,
			msg.Height-v-4,
		)
	case cmds.GetRunningContainersMsg:
		containersList := []list.Item{}

		for _, container := range msg {
			containersList = append(containersList, apptypes.ContainerListItem(container))
		}

		cmd := m.list.SetItems(containersList)
		finalCmds = append(finalCmds, cmd)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	finalCmds = append(finalCmds, cmd)

	return m, tea.Batch(finalCmds...)
}

func (m ServicesListModel) View() tea.View {
	wrapper := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true).
		BorderForeground(appstyles.PrimaryColor).
		Padding(0, 1)

	renderedList := wrapper.Render(m.list.View())

	v := tea.NewView(renderedList)
	return v
}

/*
 * Initializer function
 */

func ServicesList(items []list.Item, width int, height int) tea.Model {
	servicesList := list.New(items, customDelegate{}, width, height)
	servicesList.SetShowTitle(false)
	servicesList.SetShowHelp(false)
	servicesList.SetShowStatusBar(false)
	servicesList.Paginator.ActiveDot = " ● "
	servicesList.Paginator.InactiveDot = " ○ "

	return ServicesListModel{
		list: servicesList,
	}
}
