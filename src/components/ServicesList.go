package components

import (
	"image/color"
	"stack-stitcher/src/appstyles"

	"fmt"
	"io"
	"stack-stitcher/src/apptypes"
	"stack-stitcher/src/cmds"
	"stack-stitcher/src/constants"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"

	"charm.land/lipgloss/v2"
)

/*
 * Styling by creating a custom delegate
 */

type servicesListCustomDelegate struct {
	isParentFocused bool
	activeIndex     int
}

func (d servicesListCustomDelegate) Height() int                             { return 4 }
func (d servicesListCustomDelegate) Spacing() int                            { return 0 }
func (d servicesListCustomDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

// Render handles the actual drawing of the item
func (d servicesListCustomDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	// Cast the generic list.Item back to our specific ServiceListItem
	item, ok := listItem.(apptypes.ServiceListItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()
	isActive := index == d.activeIndex
	var titleColor color.Color

	if isActive {
		titleColor = appstyles.PrimaryFontColor
	} else {
		titleColor = appstyles.SecondaryFontColor
	}

	wrapperStyle := lipgloss.NewStyle().
		Width(m.Width()).
		Padding(1)

	titleStyle := lipgloss.NewStyle().
		Bold(isActive).
		Foreground(titleColor).
		Width(m.Width())

	if isActive {
		wrapperStyle = wrapperStyle.
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderLeftForeground(appstyles.PrimaryColor).
			Background(lipgloss.Color("#3F3F3F"))

	} else if isSelected && d.isParentFocused {
		wrapperStyle = wrapperStyle.
			Bold(true).
			BorderLeft(true).
			BorderStyle(lipgloss.DoubleBorder()).
			BorderLeftForeground(appstyles.PrimaryFontColor)

	} else {
		// Default unselected, inactive state
		wrapperStyle = wrapperStyle.
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderLeftForeground(appstyles.SecondaryFontColor)

	}

	title := titleStyle.Render(item.Title())
	description := item.Description(isActive)

	// Print the styled string to the Bubble Tea io.Writer
	fmt.Fprint(w, wrapperStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, description)))
}

/*
 * Implementation of tea.Model
 */

type ServicesListModel struct {
	list         list.Model
	listDelegate servicesListCustomDelegate
	isFocused    bool
	componentId  int
	fileName     string
	project      *types.Project
}

func (m ServicesListModel) Init() tea.Cmd {
	return nil
}

func (m ServicesListModel) Update(msg tea.Msg) (ServicesListModel, tea.Cmd) {
	var finalCmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appstyles.DocStyle.GetFrameSize()
		totalWidth := float32(msg.Width - h)
		calculatedWidth := int(totalWidth*constants.LEFT_PANEL_WIDTH - 1)
		panelWidth := max(constants.MIN_PANEL_WIDTH, calculatedWidth)

		m.list.SetSize(
			panelWidth,
			msg.Height-v-6,
		)

	case tea.KeyPressMsg:
		switch msg.String() {
		case "space":
			if m.isFocused {
				m.listDelegate.activeIndex = m.list.GlobalIndex()
				m.list.SetDelegate(m.listDelegate)

				selectedItem := m.list.SelectedItem()
				selectedService, ok := selectedItem.(apptypes.ServiceListItem)

				if ok {
					selectedServiceCmd := cmds.SetSelectedService(selectedService.Service)
					finalCmds = append(finalCmds, selectedServiceCmd)
				}
			}
		}

	case cmds.SetServicesListMsg:
		servicesList := []list.Item{}

		for _, service := range msg {
			newService := apptypes.ServiceListItem{
				Service: service,
			}

			servicesList = append(servicesList, newService)
		}

		cmd := m.list.SetItems(servicesList)
		finalCmds = append(finalCmds, cmd)

	case cmds.SetFocusMsg:
		if int(msg) == m.componentId {
			m.isFocused = true
			m.listDelegate.isParentFocused = true
			m.list.SetDelegate(m.listDelegate)
		} else {
			m.isFocused = false
			m.listDelegate.isParentFocused = false
			m.list.SetDelegate(m.listDelegate)
		}
	}

	if m.isFocused {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		finalCmds = append(finalCmds, cmd)
	}

	return m, tea.Batch(finalCmds...)
}

func (m ServicesListModel) View() tea.View {
	wrapper := lipgloss.NewStyle().
		Padding(1, 2, 2, 2)

	if m.isFocused {
		wrapper = wrapper.
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(appstyles.PrimaryColor).
			Padding(0, 1, 1, 1)
	}

	renderedList := wrapper.Render(m.list.View())

	v := tea.NewView(renderedList)
	return v
}

/*
 * Initializer function
 */

func ServicesList(services []types.ServiceConfig, width int, height int) ServicesListModel {
	var items []list.Item

	for _, service := range services {
		items = append(items, apptypes.ServiceListItem{Service: service})
	}

	listDelegate := servicesListCustomDelegate{}
	servicesList := list.New(items, listDelegate, width, height)
	servicesList.SetShowHelp(false)
	servicesList.SetShowStatusBar(false)

	servicesList.Title = "Services"
	servicesList.Paginator.ActiveDot = " ● "
	servicesList.Paginator.InactiveDot = " ○ "
	servicesList.Styles.Title = servicesList.
		Styles.
		Title.
		Background(appstyles.PrimaryColor)

	model := ServicesListModel{
		list:         servicesList,
		listDelegate: listDelegate,
		componentId:  1,
	}

	return model
}
