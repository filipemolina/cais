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

	"charm.land/lipgloss/v2"
)

/*
 * Styling by creating a custom delegate
 */

type GroupsListCustomDelegate struct {
	isParentFocused bool
	activeIndex     int
}

func (d GroupsListCustomDelegate) Height() int                             { return 4 }
func (d GroupsListCustomDelegate) Spacing() int                            { return 0 }
func (d GroupsListCustomDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

// Render handles the actual drawing of the item
func (d GroupsListCustomDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	// Cast the generic list.Item back to our specific GroupListItem
	item, ok := listItem.(apptypes.GroupListItem)
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

	// Seal the row against its own background before handing it to the list, so
	// the active row keeps its lighter surface color instead of being flattened
	// to the panel's when the list is sealed - see appstyles.FillBackground.
	row := appstyles.FillBackground(rowBg, wrapperStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title)))

	// Print the styled string to the Bubble Tea io.Writer
	fmt.Fprint(w, row)
}

/*
 * Implementation of tea.Model
 */

type GroupListModel struct {
	list         list.Model
	listDelegate GroupsListCustomDelegate
	// activeGroup is the name of the group the user picked with space. See
	// ServicesListModel.activeService - creating or deleting a group
	// reshuffles this list, so a stored row number would highlight whichever
	// group moved into that row.
	activeGroup string
	isFocused   bool
	componentId int
	statsHeader string
	panelWidth  int
	panelHeight int
}

// syncActiveIndex points the delegate at the row holding activeGroup, or at
// no row at all when it isn't in the list. Runs on both the list and the
// selection changing, since they arrive as unordered separate messages.
func (m *GroupListModel) syncActiveIndex() {
	active := -1

	for i, item := range m.list.Items() {
		if group, ok := item.(apptypes.GroupListItem); ok && string(group) == m.activeGroup {
			active = i
			break
		}
	}

	m.listDelegate.activeIndex = active
	m.list.SetDelegate(m.listDelegate)
}

func (m GroupListModel) Init() tea.Cmd {
	return nil
}

// headerHeight is the rows the stats header takes above the list.
func (m GroupListModel) headerHeight() int {
	if m.statsHeader == "" {
		return 0
	}

	return 1
}

// resizeList sizes the inner list to the space left inside the panel box
// after the wrapper padding and the stats header. Called whenever either the
// box or the header changes.
func (m *GroupListModel) resizeList() {
	h, v := listWrapperStyle.GetFrameSize()

	m.list.SetSize(
		max(0, m.panelWidth-h),
		max(0, m.panelHeight-v-m.headerHeight()),
	)
}

func (m GroupListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var finalCmds []tea.Cmd

	switch msg := msg.(type) {
	// The panel's box comes from AppModel; the inner list is sized to what
	// is left inside the wrapper's padding.
	case cmds.SetBodyLayoutMsg:
		m.panelWidth = msg.LeftWidth
		m.panelHeight = msg.Height
		m.resizeList()

	case tea.KeyPressMsg:
		switch msg.String() {
		case "space":
			if m.isFocused {
				selectedItem := m.list.SelectedItem()
				selectedGroup, ok := selectedItem.(apptypes.GroupListItem)

				if ok {
					// Highlight on the same frame as the keypress rather
					// than waiting for the message to come back around.
					m.activeGroup = string(selectedGroup)
					m.syncActiveIndex()

					selectedServiceCmd := cmds.SetSelectedGroup(string(selectedGroup))
					finalCmds = append(finalCmds, selectedServiceCmd)
				}
			}

		case "n":
			if m.isFocused {
				finalCmds = append(finalCmds, cmds.OpenCreateGroupModal())
			}

		case "d":
			if m.isFocused {
				if selectedGroup, ok := m.list.SelectedItem().(apptypes.GroupListItem); ok {
					finalCmds = append(finalCmds, cmds.OpenDeleteGroupModal(string(selectedGroup)))
				}
			}
		}

	case cmds.SetHomeStatsMsg:
		m.statsHeader = fmt.Sprintf("%d groups · %d services · %d running", msg.Groups, msg.Services, msg.Running)
		// The header appearing takes a row away from the list.
		m.resizeList()

	// AppModel decides which group is selected after a config reload, so the
	// list follows that decision rather than keeping its own.
	case cmds.SetSelectedGroupMsg:
		m.activeGroup = string(msg)
		m.syncActiveIndex()

	case cmds.SetGroupsListMsg:
		groupsList := []list.Item{}

		for _, group := range msg {
			newGroup := apptypes.GroupListItem(group)

			groupsList = append(groupsList, newGroup)
		}

		cmd := m.list.SetItems(groupsList)
		finalCmds = append(finalCmds, cmd)
		m.syncActiveIndex()

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

func (m GroupListModel) View() tea.View {
	// 3-tier background system: tier 3 (panel) when unfocused,
	// tier 4 (elevated) when focused. The focus state is shown by the
	// background lifting, not by a border.
	bg := panelBg(m.isFocused)

	// The panel fills exactly the box AppModel handed it, so the tier-3
	// background covers the full body region and both panels are the same
	// height regardless of how much content they hold.
	wrapper := fitBox(listWrapperStyle.Background(bg), m.panelWidth, m.panelHeight)

	// Rows left for the sections below, inside the wrapper padding.
	frameW, frameH := listWrapperStyle.GetFrameSize()
	contentWidth := max(0, m.panelWidth-frameW)
	contentHeight := max(0, m.panelHeight-frameH)

	var sections []string

	if m.statsHeader != "" {
		headerStyle := lipgloss.NewStyle().
			Foreground(appstyles.TextDim).
			Background(bg).
			Padding(0, 1)
		sections = append(sections, fitBox(headerStyle, contentWidth, 0).Render(m.statsHeader))
	}

	if len(m.list.Items()) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(appstyles.TextMuted).
			Background(bg).
			Padding(2, 2)
		// Width-constrained so the hint wraps inside the panel instead of
		// widening it past its box.
		sections = append(sections, fitBox(emptyStyle, contentWidth, max(0, contentHeight-m.headerHeight())).Render(
			"No groups yet.\nPress n to create one, or add profiles to services in your compose file.",
		))
	} else {
		sections = append(sections, m.list.View())
	}

	// JoinVertical pads the shorter of the stats header / list out to the
	// widest with unstyled spaces, so seal the joined block against the panel
	// tier. Rows arrive already sealed against their own background.
	content := appstyles.FillBackground(bg, lipgloss.JoinVertical(lipgloss.Left, sections...))

	v := tea.NewView(wrapper.Render(content))
	return v
}

/*
 * Initializer function
 */

func GroupsList(groups []string, width int, height int) tea.Model {
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

	servicesList.Title = "Groups"
	servicesList.Paginator.ActiveDot = " ● "
	servicesList.Paginator.InactiveDot = " ○ "
	servicesList.Styles.Title = servicesList.
		Styles.
		Title.
		Background(appstyles.PrimaryColor)

	model := GroupListModel{
		list:         servicesList,
		listDelegate: listDelegate,
		componentId:  1,
	}

	return model
}
