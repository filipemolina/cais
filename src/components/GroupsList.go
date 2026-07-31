package components

import (
	"fmt"
	"image/color"
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
		titleColor = appstyles.Active.TextPrimary
	} else {
		titleColor = appstyles.Active.TextMuted
	}

	rowBg := listRowBg(isActive, d.isParentFocused)

	// The row's left edge is the same solid bar the nav uses for its active
	// tab ("▌"), so list rows and the nav agree on thickness. State is carried
	// by color alone: accent = cursor row, primary = selected, muted = default.
	barColor := appstyles.Active.TextMuted

	if isActive {
		barColor = appstyles.Active.Accent
	} else if isSelected && d.isParentFocused {
		barColor = appstyles.Active.TextPrimary
	}

	wrapperStyle := lipgloss.NewStyle().
		Width(m.Width() - 1).
		Padding(1).
		Background(rowBg)

	// The title style only bolds the active row; the selected row's bold comes
	// from the wrapper, which is why it is applied here rather than in the title.
	if isSelected && d.isParentFocused && !isActive {
		wrapperStyle = wrapperStyle.Bold(true)
	}

	titleStyle := lipgloss.NewStyle().
		Bold(isActive).
		Foreground(titleColor).
		Background(rowBg).
		Width(m.Width() - 1)

	title := titleStyle.Render(item.Title())

	content := wrapperStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title))

	// The bar spans the row's full height, one ▌ per line, rather than a sliver
	// at the top - the nav's single-line bar stretched to the row's height.
	bar := barColumn(barColor, rowBg, content)

	// Seal the row against its own background before handing it to the list, so
	// the active row keeps its lighter surface color instead of being flattened
	// to the panel's when the list is sealed - see appstyles.FillBackground.
	row := appstyles.FillBackground(rowBg, lipgloss.JoinHorizontal(lipgloss.Left, bar, content))

	// Print the styled string to the Bubble Tea io.Writer
	fmt.Fprint(w, row)
}

/*
 * Implementation of tea.Model
 */

type GroupListModel struct {
	list         list.Model
	listDelegate GroupsListCustomDelegate
	// activeGroup is the name of the group the cursor is on. The cursor
	// position IS the selection now (auto-select on navigation), so a stored
	// row number would highlight whichever group moved into that row.
	activeGroup string
	isFocused   bool
	componentId int
	stats       cmds.SetHomeStatsMsg
	hasStats    bool
	panelWidth  int
	panelHeight int
}

// statsLine is the counts footer, in the longest form that fits `width`
// columns. It has to fit on one row: it is the panel's last line, so wrapping
// it eats into the padding below instead of just pushing the list down.
func statsLine(stats cmds.SetHomeStatsMsg, width int) string {
	full := fmt.Sprintf("%d %s · %d %s · %d running",
		stats.Groups, plural(stats.Groups, "group"),
		stats.Services, plural(stats.Services, "service"),
		stats.Running)
	if lipgloss.Width(full) <= width {
		return full
	}

	return fmt.Sprintf("%d grp · %d svc · %d run", stats.Groups, stats.Services, stats.Running)
}

// plural is the naive English plural of word for n: enough for the handful of
// countable nouns the UI puts in front of a number.
func plural(n int, word string) string {
	if n == 1 {
		return word
	}

	return word + "s"
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

// OwnsKeyboard reports whether the list is taking every keystroke for itself,
// which it does while the user is typing a filter: n, d and q are letters then,
// not commands. Only while typing - once a filter is applied and the cursor is
// back in the rows, the panel keys mean what they always mean, and esc clears
// the filter. See model.AppModel.keyboardOwned.
func (m GroupListModel) OwnsKeyboard() bool {
	return m.list.FilterState() == list.Filtering
}

// KeepsEsc reports whether the list needs esc for itself: an applied filter
// is cleared by esc alone, and the key only reaches the list while the list
// is focused. AppModel's "back" checks this before it takes focus away - see
// model.AppModel.escKept.
func (m GroupListModel) KeepsEsc() bool {
	return m.isFocused && m.list.FilterState() == list.FilterApplied
}

// FilterState exposes how much of the keyboard the list has taken, so
// AppModel can snapshot it into the help overlay's context.
func (m GroupListModel) FilterState() list.FilterState {
	return m.list.FilterState()
}

// footerHeight is the rows the stats line takes below the list.
func (m GroupListModel) footerHeight() int {
	if !m.hasStats {
		return 0
	}

	return 1
}

// resizeList sizes the inner list to the space left inside the panel box
// after the wrapper padding and the stats footer. Called whenever either the
// box or the footer changes.
func (m *GroupListModel) resizeList() {
	h, v := listWrapperStyle.GetFrameSize()

	m.list.SetSize(
		max(0, m.panelWidth-h),
		max(0, m.panelHeight-v-m.footerHeight()),
	)
}

func (m GroupListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var finalCmds []tea.Cmd

	// The footer advertises different keys depending on this, so the transition
	// has to be announced. Taken before the inner list sees the message.
	filterStateBefore := m.list.FilterState()

	switch msg := msg.(type) {
	// The panel's box comes from AppModel; the inner list is sized to what
	// is left inside the wrapper's padding.
	case cmds.SetBodyLayoutMsg:
		m.panelWidth = msg.LeftWidth
		m.panelHeight = msg.Height
		m.resizeList()

	case tea.KeyPressMsg:
		// The inner list still gets the key below - that is where the filter
		// input lives - but none of the panel's own verbs fire while it is
		// being typed into.
		if !m.isFocused || m.OwnsKeyboard() {
			break
		}

		switch {
		case key.Matches(msg, keys.List.Select):
			// Space/Enter starts the selected item (quick action).
			// Selection happens automatically on cursor movement.
			if m.activeGroup != "" {
				finalCmds = append(finalCmds, cmds.RequestDockerAction("start", m.activeGroup, true))
			}

		case key.Matches(msg, keys.List.Delete):
			if selectedGroup, ok := m.list.SelectedItem().(apptypes.GroupListItem); ok {
				finalCmds = append(finalCmds, cmds.OpenDeleteGroupModal(string(selectedGroup)))
			}

		case key.Matches(msg, keys.List.Edit):
			if selectedGroup, ok := m.list.SelectedItem().(apptypes.GroupListItem); ok {
				finalCmds = append(finalCmds, cmds.OpenEditGroupModal(string(selectedGroup)))
			}

		case key.Matches(msg, keys.List.Rename):
			if selectedGroup, ok := m.list.SelectedItem().(apptypes.GroupListItem); ok {
				finalCmds = append(finalCmds, cmds.OpenRenameGroupModal(string(selectedGroup)))
			}
		}

	case cmds.SetHomeStatsMsg:
		m.stats = msg
		m.hasStats = true
		// The footer appearing takes a row away from the list.
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
		// Track cursor before the list processes the key, so we can detect
		// movement and auto-select the item under it.
		previousIndex := m.list.Index()

		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		finalCmds = append(finalCmds, cmd)

		// Auto-select: if the cursor moved, select the item under it.
		if m.list.Index() != previousIndex {
			if item := m.list.SelectedItem(); item != nil {
				if group, ok := item.(apptypes.GroupListItem); ok {
					m.activeGroup = string(group)
					m.syncActiveIndex()
					finalCmds = append(finalCmds, cmds.SetSelectedGroup(string(group)))
				}
			}
		}
	}

	if state := m.list.FilterState(); state != filterStateBefore {
		finalCmds = append(finalCmds, cmds.SetListFilterState(state))
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

	if len(m.list.Items()) == 0 {
		// Render the list title even when empty, using the same accent-chip
		// style as the right panel's Details title (renderPanelFrame). The
		// MarginLeft(2) matches the gutter the bubbles list TitleBar adds, so
		// the empty state's chip lines up with the non-empty one's.
		titleRow := appstyles.NormalTitle().MarginLeft(2).Render(m.list.Title)
		emptyStyle := lipgloss.NewStyle().
			Foreground(appstyles.Active.TextMuted).
			Background(bg).
			Padding(2, 2)
		// Width-constrained so the hint wraps inside the panel instead of
		// widening it past its box.
		emptyContent := fitBox(emptyStyle, contentWidth, max(0, contentHeight-m.footerHeight())).Render(
			"No groups yet.\nPress n to create one, or add profiles to services in your compose file.",
		)
		sections = append(sections, appstyles.FillBackground(bg, lipgloss.JoinVertical(lipgloss.Left, titleRow, emptyContent)))
	} else {
		// The title chip is restyled here, on a copy, rather than in the
		// constructor - see appstyles.NormalTitle for why.
		l := m.list
		l.Styles.Title = appstyles.NormalTitle()
		sections = append(sections, l.View())
	}

	// The stats sit on the panel's last row rather than above the list title:
	// as a header they crowded the title chip, and they read as a summary of
	// what is above them anyway. The list fills the height it was given, so
	// appending the line here pins it to the bottom of the panel.
	if m.hasStats {
		footerStyle := lipgloss.NewStyle().
			Foreground(appstyles.Active.TextDim).
			Background(bg).
			Padding(0, 1)

		frameW, _ := footerStyle.GetFrameSize()
		sections = append(sections, fitBox(footerStyle, contentWidth, 0).Render(statsLine(m.stats, contentWidth-frameW)))
	}

	// JoinVertical pads the shorter of the stats footer / list out to the
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
	// Without this the list keeps list.DefaultKeyMap, which claims d, f, l, h,
	// b, u, q, esc and ? - keys this app spends elsewhere. See keys.ListKeyMap.
	servicesList.KeyMap = keys.ListKeyMap()

	servicesList.Title = "Groups"
	servicesList.Paginator.ActiveDot = " ● "
	servicesList.Paginator.InactiveDot = " ○ "

	model := GroupListModel{
		list:         servicesList,
		listDelegate: listDelegate,
		componentId:  1,
	}

	return model
}
