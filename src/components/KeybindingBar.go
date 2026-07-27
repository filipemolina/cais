package components

import (
	"fmt"
	"image/color"
	"path/filepath"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
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
	composeFile       string
	// composeFileOthers is how many candidates lost to composeFile. The
	// winner is the whole story only when it is the only one, so a +N marks
	// the rest; the help overlay names them.
	composeFileOthers int
	filterState       list.FilterState
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

	case cmds.SetComposeFileMsg:
		m.composeFile = msg.Name
		m.composeFileOthers = len(msg.Others)

	case cmds.SetListFilterStateMsg:
		m.filterState = list.FilterState(msg)
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
		Filter:    m.filterState,
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

// composeFileSegment renders which compose file the app resolved, dimmed, to
// sit immediately left of the global keys. Docker picks the file itself, so
// this reports rather than configures - see cmds.SetComposeFileMsg.
//
// spare is the room left over once the hints on both sides have taken theirs.
// The keys matter more than the file name, so the name degrades instead of
// pushing them off the bar: full path, then basename, then dropped.
func (m KeybindingBarModel) composeFileSegment(spare int) string {
	name := m.composeFile
	if name == "" {
		name = "no compose file"
	}

	// The count travels with the name through the degradation ladder: it is
	// part of the answer to "which file?", not extra detail to shed.
	suffix := ""
	if m.composeFileOthers > 0 {
		suffix = fmt.Sprintf(" +%d", m.composeFileOthers)
	}

	// The separator travels with the name so dropping one drops both.
	const separator = " · "

	candidates := []string{name}
	if base := filepath.Base(name); base != name {
		candidates = append(candidates, base)
	}

	style := lipgloss.NewStyle().Foreground(appstyles.TextDim)

	for _, candidate := range candidates {
		segment := candidate + suffix + separator
		if lipgloss.Width(segment) <= spare {
			return style.Render(segment)
		}
	}

	return ""
}

func (m KeybindingBarModel) View() tea.View {
	hints := m.hintsFor()

	// The page keys are global, so they sit on the right with quit rather
	// than in the context-dependent hints. The nav renders each tab's own
	// digit; this is the reminder that the digits switch pages.
	rightHint := renderKeyHints(hintsFrom(keys.Globals()), appstyles.TextDim)

	width := m.terminalWidth
	if width <= 0 {
		width = 80
	}

	left := renderKeyHints(hints, appstyles.TextDim)

	// The 4 is the bar's horizontal padding; the 1 is the narrowest gap the
	// two hint groups will accept between them. What survives that is what the
	// file name has to fit into.
	fixed := lipgloss.Width(left) + lipgloss.Width(rightHint) + 4
	file := m.composeFileSegment(width - fixed - 1)

	gapWidth := width - fixed - lipgloss.Width(file)
	if gapWidth < 1 {
		gapWidth = 1
	}
	gap := lipgloss.NewStyle().Width(gapWidth).Render("")

	bar := lipgloss.JoinHorizontal(lipgloss.Left, left, gap, file, rightHint)

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
