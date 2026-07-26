package components

import (
	"fmt"
	"image/color"
	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/cmds"
	"stack-stitcher/src/constants"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// KeyHint represents a single keybinding for display in the bottom bar.
// Key is the literal key (e.g. "n", "space", "←/→"). Desc is a short
// verb describing what the key does.
type KeyHint struct {
	Key  string
	Desc string
}

// KeybindingBar is a single-line footer that shows the current page, the
// focused component, and the keys available in that context. It listens for
// SetFocusMsg and SetActivePageMsg to track state — no direct coupling to
// the AppModel.
type KeybindingBarModel struct {
	focusedComponent  int
	activePage        string
	terminalWidth     int
	selectedGroup     string
	selectedService   bool
	groupsListEmpty   bool
	servicesListEmpty bool
}

func (m KeybindingBarModel) Init() tea.Cmd { return nil }

func (m KeybindingBarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width

	case cmds.SetFocusMsg:
		m.focusedComponent = int(msg)

	case cmds.SetActivePageMsg:
		m.activePage = string(msg)

	case cmds.SetSelectedGroupMsg:
		m.selectedGroup = string(msg)

	case cmds.SetSelectedServiceMsg:
		m.selectedService = true

	case cmds.SetGroupsListMsg:
		m.groupsListEmpty = len(msg) == 0
		if m.groupsListEmpty {
			m.selectedGroup = ""
		}

	case cmds.SetServicesListMsg:
		m.servicesListEmpty = len(msg) == 0
		if m.servicesListEmpty {
			m.selectedService = false
		}
	}
	return m, nil
}

// hintsFor returns the keybinding hints for the current page and focused
// component. The page-aware mapping keeps the bar accurate as the user
// tabs through components and switches pages.
func (m KeybindingBarModel) hintsFor() []KeyHint {
	switch m.activePage {
	case "Home":
		switch m.focusedComponent {
		case constants.COMPONENT_BODY_LIST: // Groups List
			hints := []KeyHint{
				{"n", "new"},
				{"↑/↓", "navigate"},
				{"tab", "next"},
			}
			if !m.groupsListEmpty {
				hints = append([]KeyHint{{"space", "select"}}, hints...)
				hints = append(hints[:2], append([]KeyHint{{"d", "delete"}}, hints[2:]...)...)
			}
			return hints
		case constants.COMPONENT_BODY_DETAILS: // Group Details
			if m.selectedGroup == "" {
				return []KeyHint{{"tab", "next"}}
			}
			return []KeyHint{
				{"s", "start"},
				{"t", "stop"},
				{"r", "restart"},
				{"p", "pull"},
				{"x", "remove"},
				{"l", "logs"},
				{"tab", "next"},
			}
		}
	case "Dashboard":
		switch m.focusedComponent {
		case constants.COMPONENT_BODY_LIST: // Services List
			hints := []KeyHint{
				{"↑/↓", "navigate"},
				{"tab", "next"},
			}
			if !m.servicesListEmpty {
				hints = append([]KeyHint{{"space", "select"}}, hints...)
			}
			return hints
		case constants.COMPONENT_BODY_DETAILS: // Service Details
			if !m.selectedService {
				return []KeyHint{{"tab", "next"}}
			}
			return []KeyHint{
				{"s", "start"},
				{"t", "stop"},
				{"r", "restart"},
				{"p", "pull"},
				{"x", "remove"},
				{"l", "logs"},
				{"e", "edit"},
				{"E", "file"},
				{"tab", "next"},
			}
		}

	// Placeholder pages hold nothing focusable, so offering "tab next" there
	// would advertise a key that visibly does nothing.
	case "Compose Files", "Settings":
		return nil
	}

	return []KeyHint{{"tab", "next"}}
}

// renderKeyHints renders hints as "key desc · key desc": the key bold in the
// primary text color, the description in descColor. Modals render their own
// help lines through this so they read the same as the footer bar, passing a
// lighter descColor when they sit on a lighter surface than the bar does.
func renderKeyHints(hints []KeyHint, descColor color.Color) string {
	descStyle := lipgloss.NewStyle().Foreground(descColor)
	sepStyle := lipgloss.NewStyle().Foreground(appstyles.TextDim)
	keyStyle := lipgloss.NewStyle().Foreground(appstyles.TextPrimary).Bold(true)

	parts := make([]string, 0, len(hints)*2)
	for i, h := range hints {
		if i > 0 {
			parts = append(parts, sepStyle.Render(" · "))
		}
		parts = append(parts, fmt.Sprintf("%s %s", keyStyle.Render(h.Key), descStyle.Render(h.Desc)))
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

func (m KeybindingBarModel) View() tea.View {
	hints := m.hintsFor()

	dimStyle := lipgloss.NewStyle().Foreground(appstyles.TextDim)
	keyStyle := lipgloss.NewStyle().Foreground(appstyles.TextPrimary).Bold(true)

	// The page chords are global, so they sit on the right with quit rather
	// than in the context-dependent hints. The nav underlines which letter
	// belongs to which page; this is the reminder that alt is the modifier.
	rightHint := fmt.Sprintf("%s %s%s%s %s",
		keyStyle.Render("alt+·"), dimStyle.Render("page"),
		dimStyle.Render(" · "),
		keyStyle.Render("q"), dimStyle.Render("quit"),
	)

	width := m.terminalWidth
	if width <= 0 {
		width = 80
	}

	left := renderKeyHints(hints, appstyles.TextDim)
	gapWidth := width - lipgloss.Width(left) - lipgloss.Width(rightHint) - 4
	if gapWidth < 1 {
		gapWidth = 1
	}
	gap := lipgloss.NewStyle().Width(gapWidth).Render("")

	bar := lipgloss.JoinHorizontal(lipgloss.Left, left, gap, rightHint)

	rendered := lipgloss.NewStyle().
		Background(appstyles.BackgroundContent).
		Width(width).
		Padding(0, 2).
		Render(bar)

	return tea.NewView(rendered)
}

func KeybindingBar() tea.Model {
	return KeybindingBarModel{
		focusedComponent:  constants.COMPONENT_BODY_LIST,
		activePage:        "Home",
		groupsListEmpty:   true,
		servicesListEmpty: true,
	}
}
