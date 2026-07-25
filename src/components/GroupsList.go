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

	// Print the styled string to the Bubble Tea io.Writer
	fmt.Fprint(w, wrapperStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title)))
}

/*
 * Implementation of tea.Model
 */

type GroupListModel struct {
	list         list.Model
	listDelegate GroupsListCustomDelegate
	isFocused    bool
	componentId  int
	statsHeader  string
	panelWidth   int
	panelHeight  int
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
				m.listDelegate.activeIndex = m.list.GlobalIndex()
				m.list.SetDelegate(m.listDelegate)

				selectedItem := m.list.SelectedItem()
				selectedGroup, ok := selectedItem.(apptypes.GroupListItem)

				if ok {
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

	case cmds.SetGroupsListMsg:
		groupsList := []list.Item{}

		for _, group := range msg {
			newGroup := apptypes.GroupListItem(group)

			groupsList = append(groupsList, newGroup)
		}

		cmd := m.list.SetItems(groupsList)
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

func (m GroupListModel) View() tea.View {
	// 3-tier background system: tier 3 (panel) when unfocused,
	// tier 4 (elevated) when focused. The focus state is shown by the
	// background lifting, not by a border.
	bg := appstyles.BackgroundPanel
	if m.isFocused {
		bg = appstyles.BackgroundElevated
	}

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

	v := tea.NewView(wrapper.Render(lipgloss.JoinVertical(lipgloss.Left, sections...)))
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

	listDelegate := GroupsListCustomDelegate{}
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
