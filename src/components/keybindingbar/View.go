package keybindingbar

import (
	"fmt"
	"path/filepath"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

// hintsFor returns the keybinding hints for the current page and focused
// component. Which keys are live is keys.Active's decision, not the bar's: the
// bar only supplies the screen state that decision needs, so the footer and the
// handlers cannot disagree about what is pressable.
func (m Model) hintsFor() []chrome.KeyHint {
	listEmpty := m.groupsListEmpty
	selected := m.selectedGroup != ""

	if m.activePage == "Services" {
		listEmpty = m.servicesListEmpty
		selected = m.selectedService
	}

	return hintsFrom(keys.Active(keys.Context{
		Page:          m.activePage,
		Focused:       m.focusedComponent,
		ListEmpty:     listEmpty,
		Selected:      selected,
		Editing:       m.editing,
		PendingAction: m.pendingAction,
		Filter:        m.filterState,
	}))
}

// hintsFrom turns bindings into footer hints, using the help text each binding
// carries.
func hintsFrom(bindings []key.Binding) []chrome.KeyHint {
	hints := make([]chrome.KeyHint, 0, len(bindings))

	for _, binding := range bindings {
		hints = append(hints, chrome.HintFor(binding))
	}

	return hints
}

// composeFileSegment renders which compose file the app resolved, dimmed, to
// sit immediately left of the global keys. Docker picks the file itself, so
// this reports rather than configures - see cmds.SetComposeFileMsg.
//
// spare is the room left over once the hints on both sides have taken theirs.
// The keys matter more than the file name, so the name degrades instead of
// pushing them off the bar: full path, then basename, then dropped.
func (m Model) composeFileSegment(spare int) string {
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

	style := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim)

	for _, candidate := range candidates {
		segment := candidate + suffix + separator
		if lipgloss.Width(segment) <= spare {
			return style.Render(segment)
		}
	}

	return ""
}

func (m Model) View() tea.View {
	hints := m.hintsFor()

	// The page keys are global, so they sit on the right with quit rather
	// than in the context-dependent hints. The nav renders each tab's own
	// digit; this is the reminder that the digits switch pages.
	rightHint := chrome.RenderKeyHints(hintsFrom(keys.Globals()), appstyles.Active.TextDim)

	width := m.terminalWidth
	if width <= 0 {
		width = 80
	}

	left := chrome.RenderKeyHints(hints, appstyles.Active.TextDim)

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
		Background(appstyles.Active.BackgroundContent).
		Width(width).
		Padding(0, 2).
		Render(bar)

	return tea.NewView(rendered)
}
