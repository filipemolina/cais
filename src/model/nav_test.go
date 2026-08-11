package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/constants"
	"github.com/filipemolina/cais/src/utils"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// altKey builds the key press for an alt+<letter> chord the way a terminal
// would deliver it.
func altKey(letter rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: letter, Text: string(letter), Mod: tea.ModAlt}
}

// digitKey builds the key press for a number-row key the way a terminal
// delivers it.
func digitKey(digit rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: digit, Text: string(digit)}
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
	m := GetInitialModel(utils.ComposeSource{})

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

// The digits are the primary page scheme: 1 opens the first tab, and so on.
func TestDigitSwitchesPage(t *testing.T) {
	for i, page := range apptypes.PageTitles {
		digit := rune('1' + i)

		t.Run(page, func(t *testing.T) {
			// Start somewhere else, so the key has an actual switch to make.
			from := apptypes.PageTitles[0]
			if from == page {
				from = apptypes.PageTitles[1]
			}

			m := applyLayout(drive(startup(120, 40), cmds.SetActivePageMsg(from)))

			_, cmd := m.Update(digitKey(digit))
			if got := activePageFrom(collect(cmd)); got != page {
				t.Errorf("%c from %q switched to %q, want %q", digit, from, got, page)
			}
		})
	}
}

// Re-pressing the active page's digit is a no-op for the same reason the
// chord is: switching pages re-runs the container query and the sync.
func TestDigitForTheActivePageDoesNothing(t *testing.T) {
	m := applyLayout(startup(120, 40))

	if m.activePage != "Home" {
		t.Fatalf("precondition: expected Home to be active, got %q", m.activePage)
	}

	_, cmd := m.Update(digitKey('1'))

	if got := activePageFrom(collect(cmd)); got != "" {
		t.Errorf("1 on Home re-broadcast the page as %q", got)
	}
}

// A digit with no page behind it must fall through, and a shifted digit -
// which arrives as the punctuation on that key - is not a page key at all.
func TestDigitsWithoutAPageDoNothing(t *testing.T) {
	m := applyLayout(startup(120, 40))

	for _, stroke := range []tea.KeyPressMsg{
		digitKey('9'),
		digitKey('0'),
		{Code: '!', Text: "!"},
		{Code: '1', Text: "1", Mod: tea.ModCtrl},
	} {
		if _, cmd := m.Update(stroke); activePageFrom(collect(cmd)) != "" {
			t.Errorf("%q switched pages, but it is not a page key", stroke.Text)
		}
	}
}

// [ and ] walk the tabs in order and wrap around at both ends.
func TestBracketsStepThroughPages(t *testing.T) {
	m := applyLayout(startup(120, 40))

	step := func(stroke string) string {
		t.Helper()

		updated, cmd := m.Update(tea.KeyPressMsg{Code: rune(stroke[0]), Text: stroke})
		m = updated.(AppModel)

		page := activePageFrom(collect(cmd))
		if page != "" {
			m = drive(m, cmds.SetActivePageMsg(page))
		}

		return page
	}

	for _, want := range []string{"Services", "Compose Files", "Env"} {
		if got := step("]"); got != want {
			t.Errorf("] stepped to %q, want %q", got, want)
		}
	}
	if got := step("]"); got != "Home" {
		t.Errorf("] past the last tab stepped to %q, want it to wrap to Home", got)
	}
	if got := step("["); got != "Env" {
		t.Errorf("[ from Home stepped to %q, want it to wrap to Env", got)
	}
	if got := step("["); got != "Compose Files" {
		t.Errorf("[ stepped to %q, want Compose Files", got)
	}
}

func TestPageChangeResetsFocusToLeftPanel(t *testing.T) {
	m := applyLayout(startup(120, 40))
	rightPanel := constants.COMPONENT_BODY_DETAILS
	m = drive(m, collect(m.ChangeFocus(&rightPanel))...)

	updated, cmd := m.Update(cmds.SetActivePageMsg("Services"))
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

// While a modal is open it owns all key input, so a page key must not fire
// out from under a text field the user is typing into. The digit case is not
// hypothetical: group names can contain digits.
func TestPageKeysAreInertWhileAModalIsOpen(t *testing.T) {
	for name, stroke := range map[string]tea.KeyPressMsg{
		"chord": altKey('d'),
		"digit": digitKey('2'),
	} {
		t.Run(name, func(t *testing.T) {
			m := applyLayout(drive(startup(120, 40),
				cmds.GetConfigMsg{FileName: "compose.yaml", Project: project()},
				cmds.OpenCreateGroupModalMsg{},
			))

			if m.activeModal == nil {
				t.Fatal("precondition: expected a modal to be open")
			}

			_, cmd := m.Update(stroke)

			if got := activePageFrom(collect(cmd)); got != "" {
				t.Errorf("%v navigated to %q while a modal was open", stroke, got)
			}
		})
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

// The digit on each tab is what tells the user which key to press, so it has
// to be the tab's own position in the page list.
func TestNavRendersEachTabsDigit(t *testing.T) {
	m := applyLayout(startup(120, 40))
	nav := ansi.Strip(m.components.MainMenu.View().Content)

	for i, page := range apptypes.PageTitles {
		tab := fmt.Sprintf("%d %s", i+1, apptypes.PageLabel(page))
		if !strings.Contains(nav, tab) {
			t.Errorf("page %q: tab %q missing from the nav", page, tab)
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
