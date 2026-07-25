package components

import (
	"image/color"
	"stack-stitcher/src/appstyles"

	"fmt"
	"io"
	"stack-stitcher/src/apptypes"
	"stack-stitcher/src/cmds"

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

	rowBg := listRowBg(isActive, d.isParentFocused)

	wrapperStyle := lipgloss.NewStyle().
		Width(m.Width()).
		Padding(1).
		Background(rowBg)

	titleStyle := lipgloss.NewStyle().
		Bold(isActive).
		Foreground(titleColor).
		Background(rowBg).
		Width(m.Width())

	if isActive {
		wrapperStyle = wrapperStyle.
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderLeftForeground(appstyles.PrimaryColor)

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

	// Seal the row against its own background before handing it to the list:
	// JoinVertical pads the description out to the title's width with unstyled
	// spaces, which would otherwise show the terminal background through the
	// row. Sealing here (rather than over the whole list) is what keeps the
	// active row's lighter surface color from being flattened to the panel's.
	row := appstyles.FillBackground(rowBg, wrapperStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, description)))

	// Print the styled string to the Bubble Tea io.Writer
	fmt.Fprint(w, row)
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
	panelWidth   int
	panelHeight  int
}

func (m ServicesListModel) Init() tea.Cmd {
	return nil
}

// resizeList sizes the inner list to the space left inside the panel box
// after the wrapper padding.
func (m *ServicesListModel) resizeList() {
	h, v := listWrapperStyle.GetFrameSize()

	m.list.SetSize(
		max(0, m.panelWidth-h),
		max(0, m.panelHeight-v),
	)
}

func (m ServicesListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var finalCmds []tea.Cmd

	switch msg := msg.(type) {
	// Sizing comes from AppModel, not WindowSizeMsg: the Dashboard is never
	// the active page when the terminal is first measured, so a resize-derived
	// height left this list a few rows tall showing a single service.
	case cmds.SetBodyLayoutMsg:
		m.panelWidth = msg.LeftWidth
		m.panelHeight = msg.Height
		m.resizeList()

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
	// Same 3-tier treatment as the groups list: focus lifts the panel from
	// tier 3 to tier 4 rather than adding a border, so the panel's box stays
	// the same size whether or not it is focused.
	bg := panelBg(m.isFocused)

	wrapper := fitBox(listWrapperStyle.Background(bg), m.panelWidth, m.panelHeight)

	// The list joins its title, rows and paginator internally, padding the
	// short ones with unstyled spaces; seal them against the panel tier. Rows
	// arrive already sealed against their own background, so this only fills
	// what the list itself left bare.
	v := tea.NewView(wrapper.Render(appstyles.FillBackground(bg, m.list.View())))
	return v
}

/*
 * Initializer function
 */

func ServicesList(services []types.ServiceConfig, width int, height int) tea.Model {
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
