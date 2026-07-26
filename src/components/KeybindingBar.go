package components

import (
	"fmt"
	"image/color"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/constants"
	"github.com/filipemolina/stack-stitcher/src/keys"
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
// component. Which keys are live is keys.Active's decision, not the bar's: the
// bar only supplies the screen state that decision needs, so the footer and the
// handlers cannot disagree about what is pressable.
func (m KeybindingBarModel) hintsFor() []KeyHint {
	listEmpty := m.groupsListEmpty
	selected := m.selectedGroup != ""

	if m.activePage == "Services" {
		listEmpty = m.servicesListEmpty
		selected = m.selectedService
	}

	return hintsFrom(keys.Active(keys.Context{
		Page:      m.activePage,
		Focused:   m.focusedComponent,
		ListEmpty: listEmpty,
		Selected:  selected,
	}))
}

// hintsFrom turns bindings into footer hints, using the help text each binding
// carries.
func hintsFrom(bindings []key.Binding) []KeyHint {
	hints := make([]KeyHint, 0, len(bindings))

	for _, binding := range bindings {
		hints = append(hints, hintFor(binding))
	}

	return hints
}

// hintFor is one binding as a hint. hintAs overrides the description for the
// places where a shared key does something more specific than its general help
// text says - Enter is "confirm" everywhere, but "create group" in the
// checklist that creates a group.
func hintFor(binding key.Binding) KeyHint {
	help := binding.Help()

	return KeyHint{help.Key, help.Desc}
}

func hintAs(binding key.Binding, desc string) KeyHint {
	return KeyHint{binding.Help().Key, desc}
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

	// The page chords are global, so they sit on the right with quit rather
	// than in the context-dependent hints. The nav underlines which letter
	// belongs to which page; this is the reminder that alt is the modifier.
	rightHint := renderKeyHints(hintsFrom(keys.Globals()), appstyles.TextDim)

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
