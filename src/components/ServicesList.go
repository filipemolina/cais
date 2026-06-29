package components

import (
	"fmt"
	"io"
	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/apptypes"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

/*
 * Styling by creating a custom delegate
 */

type customDelegate struct{}

func (d customDelegate) Height() int                             { return 3 }
func (d customDelegate) Spacing() int                            { return 0 }
func (d customDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

// Render handles the actual drawing of the item
func (d customDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	// Cast the generic list.Item back to our specific myItem
	i, ok := listItem.(apptypes.ListItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()
	isActive := false

	wrapperStyle := lipgloss.NewStyle().MarginBottom(1).Width(m.Width())
	titleStyle := lipgloss.NewStyle().Bold(true).Width(m.Width())
	descriptionStyle := lipgloss.NewStyle().Foreground(appstyles.SecondaryFontColor)

	if isSelected {
		wrapperStyle = wrapperStyle.
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderLeftForeground(lipgloss.Color("211")).
			Background(lipgloss.Color("#3F3F3F")).
			PaddingLeft(1)
	} else if isActive {
		// Highlight text if active, but not currently selected
		wrapperStyle = wrapperStyle.
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderLeftForeground(appstyles.PrimaryFontColor).
			PaddingLeft(1)
	} else {
		// Default unselected, inactive state
		wrapperStyle = wrapperStyle.
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderLeftForeground(appstyles.SecondaryFontColor).
			PaddingLeft(1)
	}

	title := titleStyle.Render(i.ItemTitle)
	description := descriptionStyle.Render(i.ItemDesc)

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
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appstyles.DocStyle.GetFrameSize()

		m.list.SetSize(
			msg.Width-h,
			msg.Height-v,
		)

	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	return m, cmd
}

func (m ServicesListModel) View() tea.View {
	v := tea.NewView(m.list.View())
	return v
}

/*
 * Initializer function
 */

func ServicesList(items []list.Item, width int, height int) tea.Model {
	servicesList := list.New(items, customDelegate{}, width, height)
	servicesList.SetShowHelp(false)
	servicesList.SetShowPagination(false)
	servicesList.SetShowTitle(false)
	servicesList.SetShowStatusBar(false)

	return ServicesListModel{
		list: servicesList,
	}
}
