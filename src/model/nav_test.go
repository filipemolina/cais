package model

import (
	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/constants"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// altKey builds the key press for an alt+<letter> chord the way a terminal
// would deliver it.
func altKey(letter rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: letter, Text: string(letter), Mod: tea.ModAlt}
}

// collect drains a command, returning every message it produces. tea.Batch
// wraps its children in a BatchMsg, so a single Update can yield several.
func collect(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}

	msg := cmd()

	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, child := range batch {
			msgs = append(msgs, collect(child)...)
		}
		return msgs
	}

	return []tea.Msg{msg}
}

// activePageFrom returns the page named by a SetActivePageMsg among msgs.
func activePageFrom(msgs []tea.Msg) string {
	for _, msg := range msgs {
		if page, ok := msg.(cmds.SetActivePageMsg); ok {
			return string(page)
		}
	}

	return ""
}

// focusedComponentFrom returns the component requested by a SetFocusMsg.
func focusedComponentFrom(msgs []tea.Msg) (int, bool) {
	for _, msg := range msgs {
		if component, ok := msg.(cmds.SetFocusMsg); ok {
			return int(component), true
		}
	}

	return 0, false
}

// Init activates the first page, and page activation owns focus assignment.
// This guarantees the focus message is delivered after there is an active
// page to receive it, rather than racing the initial SetActivePageMsg.
func TestInitialPageActivationFocusesLeftPanel(t *testing.T) {
	m := GetInitialModel()

	updated, cmd := m.Update(cmds.SetActivePageMsg(apptypes.PageTitles[0]))
	m = updated.(AppModel)

	if got, want := m.focusedComponent, constants.COMPONENT_BODY_LIST; got != want {
		t.Fatalf("startup focus: got %d, want %d", got, want)
	}

	got, ok := focusedComponentFrom(collect(cmd))
	if !ok {
		t.Fatal("initial page activation did not send a focus message")
	}
	if want := constants.COMPONENT_BODY_LIST; got != want {
		t.Errorf("initial page focus: got %d, want %d", got, want)
	}
}

func TestAltLetterSwitchesPage(t *testing.T) {
	for _, page := range apptypes.PageTitles {
		letter := []rune(apptypes.PageShortcut(page))[0]

		t.Run(page, func(t *testing.T) {
			// Start somewhere else, so the chord has an actual switch to make.
			from := apptypes.PageTitles[0]
			if from == page {
				from = apptypes.PageTitles[1]
			}

			m := applyLayout(drive(startup(120, 40), cmds.SetActivePageMsg(from)))

			// Leave the current page with its details panel focused. A page
			// shortcut must reset that state for the page it opens.
			rightPanel := constants.COMPONENT_BODY_DETAILS
			m = drive(m, collect(m.ChangeFocus(&rightPanel))...)

			updated, cmd := m.Update(altKey(letter))
			m = updated.(AppModel)
			if got := activePageFrom(collect(cmd)); got != page {
				t.Errorf("alt+%c from %q switched to %q, want %q", letter, from, got, page)
			}

			// The keyboard command queues SetActivePageMsg. Process it as the
			// Bubble Tea runtime would, then inspect the command it returns.
			updated, pageCmd := m.Update(cmds.SetActivePageMsg(page))
			m = updated.(AppModel)

			if got, want := m.focusedComponent, constants.COMPONENT_BODY_LIST; got != want {
				t.Errorf("alt+%c page focus: got %d, want %d", letter, got, want)
			}

			got, ok := focusedComponentFrom(collect(pageCmd))
			if !ok {
				t.Errorf("alt+%c page switch did not send a focus message", letter)
			} else if want := constants.COMPONENT_BODY_LIST; got != want {
				t.Errorf("alt+%c page focus message: got %d, want %d", letter, got, want)
			}
		})
	}
}

func TestPageChangeResetsFocusToLeftPanel(t *testing.T) {
	m := applyLayout(startup(120, 40))
	rightPanel := constants.COMPONENT_BODY_DETAILS
	m = drive(m, collect(m.ChangeFocus(&rightPanel))...)

	updated, cmd := m.Update(cmds.SetActivePageMsg("Dashboard"))
	m = updated.(AppModel)

	if got, want := m.focusedComponent, constants.COMPONENT_BODY_LIST; got != want {
		t.Fatalf("page switch focus: got %d, want %d", got, want)
	}

	got, ok := focusedComponentFrom(collect(cmd))
	if !ok {
		t.Fatal("page switch did not send a focus message")
	}
	if want := constants.COMPONENT_BODY_LIST; got != want {
		t.Errorf("page switch focus message: got %d, want %d", got, want)
	}
}

// Re-pressing the active page's chord is a no-op: switching pages re-runs the
// container query and the services/groups sync, and there is nothing to
// re-sync if the page has not changed.
func TestAltLetterForTheActivePageDoesNothing(t *testing.T) {
	m := applyLayout(startup(120, 40))

	if m.activePage != "Home" {
		t.Fatalf("precondition: expected Home to be active, got %q", m.activePage)
	}

	_, cmd := m.Update(altKey('g'))

	if got := activePageFrom(collect(cmd)); got != "" {
		t.Errorf("alt+g on Home re-broadcast the page as %q", got)
	}
}

// A letter with no page must fall through rather than being swallowed, and a
// bare letter must not navigate - "d" is delete on the groups list.
func TestPageShortcutsRequireAlt(t *testing.T) {
	m := applyLayout(startup(120, 40))

	bare := tea.KeyPressMsg{Code: 'd', Text: "d"}
	if _, cmd := m.Update(bare); activePageFrom(collect(cmd)) != "" {
		t.Error("bare d switched pages; it should reach the focused component instead")
	}

	if _, cmd := m.Update(altKey('z')); activePageFrom(collect(cmd)) != "" {
		t.Error("alt+z switched pages, but no page is bound to z")
	}
}

// While a modal is open it owns all key input, so a page chord must not fire
// out from under a text field the user is typing into.
func TestPageShortcutsAreInertWhileAModalIsOpen(t *testing.T) {
	m := applyLayout(drive(startup(120, 40),
		cmds.GetConfigMsg{FileName: "compose.yaml", Project: project()},
		cmds.OpenCreateGroupModalMsg{},
	))

	if m.activeModal == nil {
		t.Fatal("precondition: expected a modal to be open")
	}

	_, cmd := m.Update(altKey('d'))

	if got := activePageFrom(collect(cmd)); got != "" {
		t.Errorf("alt+d navigated to %q while a modal was open", got)
	}
}

// The nav is out of the focus cycle, so Tab must alternate between the two body
// panels and never land on the menu.
func TestTabCyclesOnlyTheBodyPanels(t *testing.T) {
	m := applyLayout(startup(120, 40))

	if got, want := m.focusedComponent, constants.COMPONENT_BODY_LIST; got != want {
		t.Fatalf("startup focus: got %d, want %d", got, want)
	}

	seen := map[int]bool{m.focusedComponent: true}
	for range 6 {
		m.ChangeFocus(nil)
		seen[m.focusedComponent] = true

		if m.focusedComponent == constants.COMPONENT_MAIN_MENU {
			t.Fatal("Tab landed on the main menu, which is not focusable")
		}
	}

	for _, id := range constants.FocusableComponents {
		if !seen[id] {
			t.Errorf("Tab never reached focusable component %d", id)
		}
	}
}

func TestShiftTabWrapsBackwards(t *testing.T) {
	m := applyLayout(startup(120, 40))
	back := -1

	// From the first focusable component, Shift+Tab wraps to the last.
	m.ChangeFocus(&back)

	order := constants.FocusableComponents
	if got, want := m.focusedComponent, order[len(order)-1]; got != want {
		t.Errorf("Shift+Tab from the first component: got %d, want %d", got, want)
	}
}

// The underline is what tells the user which letter to press, so it has to be
// on the letter the chord actually uses.
func TestNavUnderlinesTheShortcutLetter(t *testing.T) {
	m := applyLayout(startup(120, 40))
	nav := m.components.MainMenu.View().Content

	for _, page := range apptypes.PageTitles {
		label := apptypes.PageLabel(page)
		first := string([]rune(label)[0])

		// SGR 4 is underline; it must open immediately before the letter.
		if !strings.Contains(nav, "4m"+first) {
			t.Errorf("page %q: first letter %q of label %q is not underlined", page, first, label)
		}

		if !strings.Contains(ansi.Strip(nav), label) {
			t.Errorf("page %q: label %q missing from the nav", page, label)
		}
	}
}

// Requested change: the wordmark moves from the far left to the far right.
func TestWordmarkSitsAtTheFarRight(t *testing.T) {
	m := applyLayout(startup(120, 40))
	nav := ansi.Strip(m.components.MainMenu.View().Content)

	firstLine := strings.SplitN(nav, "\n", 2)[0]
	trimmed := strings.TrimRight(firstLine, " ")

	if !strings.HasSuffix(trimmed, "Stack Stitcher") {
		t.Errorf("wordmark is not at the right edge of the nav: %q", firstLine)
	}

	if strings.HasPrefix(strings.TrimLeft(firstLine, " "), "▌ Stack Stitcher") {
		t.Error("wordmark is still at the left edge of the nav")
	}
}
