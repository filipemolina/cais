package keybindingbar

import (
	"fmt"
	"path/filepath"
	"slices"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/keys"
)

// bindingsFor returns the keys live in the current page and focused component.
// Which keys those are is keys.Active's decision, not the bar's: the bar only
// supplies the screen state that decision needs, so the footer and the handlers
// cannot disagree about what is pressable.
func (m Model) bindingsFor() []key.Binding {
	listEmpty := m.groupsListEmpty
	selected := m.selectedGroup != ""

	if m.activePage == "Services" {
		listEmpty = m.servicesListEmpty
		selected = m.selectedService
	}

	return keys.Active(keys.Context{
		Page:          m.activePage,
		Focused:       m.focusedComponent,
		ListEmpty:     listEmpty,
		Selected:      selected,
		Editing:       m.editing,
		PendingAction: m.pendingAction,
		Filter:        m.filterState,
	})
}

// hintsFor is bindingsFor as hints - the whole set, before the bar has decided
// what fits.
func (m Model) hintsFor() []chrome.KeyHint {
	return hintsFrom(m.bindingsFor())
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

// barChrome is the columns the bar spends on itself whatever it is showing:
// 4 of horizontal padding, plus the narrowest gap the two hint groups will
// accept between them. What survives that is what the file name has to fit
// into.
const barChrome = 4 + 1

// fits reports whether both hint groups can sit on one line of a bar `width`
// columns wide.
func fits(left, right []key.Binding, width int) bool {
	rendered := func(bindings []key.Binding) int {
		return lipgloss.Width(chrome.RenderKeyHints(hintsFrom(bindings), appstyles.Active.TextDim))
	}

	return rendered(left)+rendered(right)+barChrome <= width
}

// shedToFit drops whole hints until the bar fits on one line, lowest
// keys.Priority first and rightmost among equals - so a page gives up its
// last-listed verb before its first, and Active's reading order doubles as its
// keep order.
//
// This is the footer's half of a fix the details panels' action row had first:
// lipgloss wraps on the cell, not on the control, so a bar wider than the
// terminal broke mid-hint and spilled onto a second line, eating a row of the
// body under it. Dropping whole hints is safe because `? help` is ranked never
// to be shed: everything that leaves the bar is still in the overlay, and still
// pressable.
func shedToFit(left, right []key.Binding, width int) ([]key.Binding, []key.Binding) {
	for !fits(left, right, width) {
		// One flat pass over both sides, left to right, keeping the last
		// binding of the lowest rank seen - which is the rightmost of the
		// least important, the one the eye misses least.
		side, at, lowest := -1, -1, keys.Priority(keys.Global.Quit)

		for s, bindings := range [][]key.Binding{left, right} {
			for i, binding := range bindings {
				if p := keys.Priority(binding); p <= lowest {
					side, at, lowest = s, i, p
				}
			}
		}

		// Everything left is ranked never-shed: the bar is as short as it can
		// honestly be, and a terminal this narrow has bigger problems.
		if side == -1 || lowest == keys.Priority(keys.Global.Quit) {
			return left, right
		}

		if side == 0 {
			left = slices.Delete(slices.Clone(left), at, at+1)
		} else {
			right = slices.Delete(slices.Clone(right), at, at+1)
		}
	}

	return left, right
}

func (m Model) View() tea.View {
	width := m.terminalWidth
	if width <= 0 {
		width = 80
	}

	// The page keys are global, so they sit on the right with quit rather
	// than in the context-dependent hints. The nav renders each tab's own
	// digit; this is the reminder that the digits switch pages.
	leftKeys, rightKeys := shedToFit(m.bindingsFor(), keys.Globals(), width)

	left := chrome.RenderKeyHints(hintsFrom(leftKeys), appstyles.Active.TextDim)
	rightHint := chrome.RenderKeyHints(hintsFrom(rightKeys), appstyles.Active.TextDim)

	fixed := lipgloss.Width(left) + lipgloss.Width(rightHint) + 4
	file := m.composeFileSegment(width - fixed - 1)

	gapWidth := width - fixed - lipgloss.Width(file)
	if gapWidth < 1 {
		gapWidth = 1
	}
	gap := lipgloss.NewStyle().Width(gapWidth).Render("")

	bar := lipgloss.JoinHorizontal(lipgloss.Left, left, gap, file, rightHint)

	// MaxHeight is the backstop under shedToFit: below about 20 columns even
	// the two never-shed hints do not fit, and a wrapped footer would eat a row
	// of the body. Clipping keeps the layout intact at widths where nothing can
	// be legible anyway - the same safety cap the details panels put on their
	// own bodies.
	rendered := lipgloss.NewStyle().
		Background(appstyles.Active.BackgroundContent).
		Width(width).
		MaxHeight(1).
		Padding(0, 2).
		Render(bar)

	return tea.NewView(rendered)
}
