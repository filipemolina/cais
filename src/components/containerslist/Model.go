package containerslist

import (
	"fmt"
	"image/color"
	"io"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/constants"
)

/*
 * Styling by creating a custom delegate
 */

type containersListCustomDelegate struct {
	isParentFocused bool
}

func (d containersListCustomDelegate) Height() int                             { return 4 }
func (d containersListCustomDelegate) Spacing() int                            { return 0 }
func (d containersListCustomDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

// Render handles the actual drawing of the item
func (d containersListCustomDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	// Cast the generic list.Item back to our specific ContainerListItem
	item, ok := listItem.(apptypes.ContainerListItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()
	isActive := false

	// The row's left edge is the same solid bar the nav uses for its active
	// tab ("▌"), so list rows and the nav agree on thickness. State is carried
	// by color alone: accent = cursor row, primary = selected, muted = default.
	barColor := appstyles.Active.TextMuted

	if isSelected && d.isParentFocused {
		barColor = appstyles.Active.Accent
	} else if isActive {
		barColor = appstyles.Active.TextPrimary
	}

	wrapperStyle := lipgloss.NewStyle().
		Width(m.Width() - 1).
		Padding(1)

	// The selected row lifts to the modal surface; the bar must share that
	// background so the row renders as one solid strip, as the old border did.
	if isSelected && d.isParentFocused {
		wrapperStyle = wrapperStyle.Background(appstyles.Active.ModalBg)
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Width(m.Width() - 1)

	var barBg color.Color
	if isSelected && d.isParentFocused {
		barBg = appstyles.Active.ModalBg
	}

	title := titleStyle.Render(item.Title())
	description := item.Description(isSelected && d.isParentFocused)

	content := wrapperStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, description))

	// The bar spans the row's full height, one ▌ per line, rather than a sliver
	// at the top - the nav's single-line bar stretched to the row's height.
	bar := chrome.BarColumn(barColor, barBg, content)

	// Print the styled string to the Bubble Tea io.Writer
	fmt.Fprint(w, lipgloss.JoinHorizontal(lipgloss.Left, bar, content))
}

/*
 * Implementation of tea.Model
 */

type Model struct {
	list        list.Model
	isFocused   bool
	componentId int
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case cmds.GetRunningContainersMsg:
		containersList := []list.Item{}

		for _, container := range msg.Containers {
			containersList = append(containersList, apptypes.ContainerListItem(container))
		}

		cmd := m.list.SetItems(containersList)
		finalCmds = append(finalCmds, cmd)

	case cmds.SetFocusMsg:
		if int(msg) == m.componentId {
			m.isFocused = true
			m.list.SetDelegate(containersListCustomDelegate{isParentFocused: true})
		} else {
			m.isFocused = false
			m.list.SetDelegate(containersListCustomDelegate{isParentFocused: false})
		}
	}

	if m.isFocused {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		finalCmds = append(finalCmds, cmd)
	}

	return m, tea.Batch(finalCmds...)
}

func (m Model) View() tea.View {
	wrapper := lipgloss.NewStyle().
		Padding(1, 2, 2, 2)

	if m.isFocused {
		wrapper = wrapper.
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(appstyles.Active.Accent).
			Padding(0, 1, 1, 1)
	}

	// The title chip is restyled here, on a copy, rather than in the
	// constructor - see appstyles.NormalTitle for why.
	l := m.list
	l.Styles.Title = appstyles.NormalTitle()

	renderedList := wrapper.Render(l.View())

	v := tea.NewView(renderedList)
	return v
}

/*
 * Initializer function
 */

func New(containers []apptypes.ContainerListItem, width int, height int) tea.Model {
	var items []list.Item

	for _, container := range containers {
		items = append(items, container)
	}

	servicesList := list.New(items, containersListCustomDelegate{}, width, height)
	servicesList.SetShowHelp(false)
	servicesList.SetShowStatusBar(false)

	servicesList.Title = "Services"
	servicesList.Paginator.ActiveDot = " ● "
	servicesList.Paginator.InactiveDot = " ○ "

	return Model{
		list:        servicesList,
		componentId: 1,
	}
}
