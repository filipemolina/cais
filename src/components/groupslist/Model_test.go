package groupslist

import (
	"fmt"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/cais/src/cmds"
)

// groupNames makes n distinct group names, enough of them to force the list
// onto more than one page.
func groupNames(n int) cmds.SetGroupsListMsg {
	names := make([]string, 0, n)
	for i := range n {
		names = append(names, fmt.Sprintf("group-%02d", i))
	}

	return cmds.SetGroupsListMsg(names)
}

// focusedGroupsList is a groups list holding n groups, focused, and sized to a
// panel short enough that the groups do not all fit on one page.
func focusedGroupsList(t *testing.T, n int) Model {
	t.Helper()

	var model tea.Model = New(nil, 40, 20)
	for _, msg := range []tea.Msg{
		cmds.SetBodyLayoutMsg{LeftWidth: 40, Height: 20},
		groupNames(n),
		cmds.SetFocusMsg(1),
	} {
		model, _ = model.Update(msg)
	}

	groups, ok := model.(Model)
	if !ok {
		t.Fatalf("expected a Model, got %T", model)
	}
	if groups.list.Paginator.TotalPages < 2 {
		t.Fatalf("wanted a list of at least two pages, got %d", groups.list.Paginator.TotalPages)
	}

	return groups
}

// messagesFrom flattens what a command produced, walking batches, so a test can
// assert on a message without caring how it got bundled.
func messagesFrom(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}

	msg := cmd()

	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, inner := range batch {
			msgs = append(msgs, messagesFrom(inner)...)
		}

		return msgs
	}

	return []tea.Msg{msg}
}

func press(t *testing.T, model tea.Model, keystroke string) (tea.Model, []tea.Msg) {
	t.Helper()

	model, cmd := model.Update(tea.KeyPressMsg{Code: rune(keystroke[0]), Text: keystroke})

	return model, messagesFrom(cmd)
}

// TestDeleteKeyDoesNotAlsoPageTheList is the regression this phase exists for.
// list.DefaultKeyMap binds d to NextPage, so d used to do both jobs: open the
// delete confirm and move the list under it.
func TestDeleteKeyDoesNotAlsoPageTheList(t *testing.T) {
	groups := focusedGroupsList(t, 12)
	startPage := groups.list.Paginator.Page

	model, msgs := press(t, groups, "d")

	var opened bool
	for _, msg := range msgs {
		if _, ok := msg.(cmds.OpenDeleteGroupModalMsg); ok {
			opened = true
		}
	}
	if !opened {
		t.Errorf("d did not open the delete confirm, got %#v", msgs)
	}

	after, ok := model.(Model)
	if !ok {
		t.Fatalf("expected a Model, got %T", model)
	}
	if after.list.Paginator.Page != startPage {
		t.Errorf("d moved the list from page %d to page %d", startPage, after.list.Paginator.Page)
	}
}

// The list keeps esc only while it can use it: focused, with a filter
// standing. Unfocused or unfiltered it has no claim, and mid-typing it owns
// the whole keyboard instead.
func TestKeepsEscOnlyWhileFocusedWithAnAppliedFilter(t *testing.T) {
	groups := focusedGroupsList(t, 12)

	if groups.KeepsEsc() {
		t.Error("an unfiltered list kept esc")
	}

	apply := func(m Model, msgs ...tea.Msg) Model {
		for _, msg := range msgs {
			updated, _ := m.Update(msg)
			m = updated.(Model)
		}
		return m
	}

	filtered := apply(groups,
		tea.KeyPressMsg{Code: '/', Text: "/"},
		tea.KeyPressMsg{Code: 'g', Text: "g"},
		tea.KeyPressMsg{Code: tea.KeyEnter},
	)

	if state := filtered.list.FilterState(); state != list.FilterApplied {
		t.Fatalf("precondition: filter state is %v, want applied", state)
	}
	if !filtered.KeepsEsc() {
		t.Error("a focused list with a filter standing did not keep esc")
	}

	unfocused := apply(filtered, cmds.SetFocusMsg(2))
	if unfocused.KeepsEsc() {
		t.Error("an unfocused list kept esc, but it never sees the key")
	}
}

// Enter starts the selected group. Selection happens automatically on cursor
// movement, so we move the cursor first, then press enter.
func TestEnterStartsTheHighlightedGroup(t *testing.T) {
	groups := focusedGroupsList(t, 12)

	// Move the cursor down to trigger auto-select (cursor goes from 0 to 1).
	model, _ := groups.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	moved, ok := model.(Model)
	if !ok {
		t.Fatalf("expected a Model, got %T", model)
	}

	// Verify auto-select happened (index 1 = group-01).
	if moved.activeGroup != "group-01" {
		t.Fatalf("auto-select did not fire: activeGroup = %q", moved.activeGroup)
	}

	// Now press enter to start the group.
	model, cmd := moved.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	var started bool
	for _, msg := range messagesFrom(cmd) {
		if dockerMsg, ok := msg.(cmds.RunDockerActionMsg); ok {
			if dockerMsg.Action == "start" && dockerMsg.Target == "group-01" && dockerMsg.IsGroup {
				started = true
			}
		}
	}
	if !started {
		t.Errorf("enter did not start group-01, got %#v", messagesFrom(cmd))
	}
}

// The other letters the default map claimed, still spent elsewhere in the app
// regardless of which panel is focused: f follows a log stream, b browses
// compose files, u opens the usage overlay. None of them pages the list.
func TestPanelLettersDoNotPageTheList(t *testing.T) {
	for _, keystroke := range []string{"f", "b", "u"} {
		t.Run(keystroke, func(t *testing.T) {
			groups := focusedGroupsList(t, 12)
			startPage := groups.list.Paginator.Page

			model, _ := press(t, groups, keystroke)

			after, ok := model.(Model)
			if !ok {
				t.Fatalf("expected a Model, got %T", model)
			}
			if after.list.Paginator.Page != startPage {
				t.Errorf("%q moved the list from page %d to page %d", keystroke, startPage, after.list.Paginator.Page)
			}
		})
	}
}

// h and l page the list like the arrow keys do - they mean nothing else while
// the list itself has focus, so they get the vim treatment left/right already
// have in k/j's up/down.
func TestVimLettersPageTheList(t *testing.T) {
	tests := []struct {
		keystroke string
		wantDelta int
	}{
		{"l", 1},
		{"h", -1},
	}

	for _, test := range tests {
		t.Run(test.keystroke, func(t *testing.T) {
			groups := focusedGroupsList(t, 12)
			// Land on page 1 first so "h" (prev page) has somewhere to go.
			groups.list.Paginator.Page = 1
			startPage := groups.list.Paginator.Page

			model, _ := press(t, groups, test.keystroke)

			after, ok := model.(Model)
			if !ok {
				t.Fatalf("expected a Model, got %T", model)
			}
			if got := after.list.Paginator.Page; got != startPage+test.wantDelta {
				t.Errorf("%q moved the list from page %d to page %d, want %d", test.keystroke, startPage, got, startPage+test.wantDelta)
			}
		})
	}
}

// The ungrouped note only means anything once groups exist to be left out
// of - with none yet, every service is ungrouped by definition, so the
// count would just repeat Services rather than warn about anything.
func TestStatsLineOmitsUngroupedUntilAGroupExists(t *testing.T) {
	stats := cmds.SetHomeStatsMsg{Groups: 0, Services: 3, Running: 0, Ungrouped: 3}

	got := statsLine(stats, 80)
	if got != "0 groups · 3 services · 0 running" {
		t.Errorf("statsLine with no groups: got %q", got)
	}
}

func TestStatsLineNotesUngroupedServices(t *testing.T) {
	stats := cmds.SetHomeStatsMsg{Groups: 1, Services: 3, Running: 1, Ungrouped: 2}

	got := statsLine(stats, 80)
	want := "1 group · 3 services · 1 running · 2 ungrouped, always run"
	if got != want {
		t.Errorf("statsLine with ungrouped services:\n got: %q\nwant: %q", got, want)
	}
}

// A narrow terminal sheds the ungrouped note before it sheds the abbreviated
// form of the core three counts - the same "shed whole things" ladder
// docs/DESIGN.md documents for the footer and the member table: the note is
// a courtesy, not the panel's own verb, so it goes first.
func TestStatsLineShedsTheUngroupedNoteFirst(t *testing.T) {
	stats := cmds.SetHomeStatsMsg{Groups: 1, Services: 3, Running: 1, Ungrouped: 2}

	full := "1 group · 3 services · 1 running · 2 ungrouped, always run"
	short := "1 grp · 3 svc · 1 run · 2 ungrp"
	shortCore := "1 grp · 3 svc · 1 run"

	tests := []struct {
		width int
		want  string
	}{
		{80, full},
		{lipgloss.Width(short), short},
		{lipgloss.Width(short) - 1, shortCore},
	}

	for _, tc := range tests {
		if got := statsLine(stats, tc.width); got != tc.want {
			t.Errorf("statsLine at width %d: got %q, want %q", tc.width, got, tc.want)
		}
	}
}

// t stops the highlighted group, the other half of the quick-action pair
// Enter/Space starts with - no Tab to the details panel required.
func TestStopStopsTheHighlightedGroup(t *testing.T) {
	groups := focusedGroupsList(t, 12)

	model, _ := groups.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	moved, ok := model.(Model)
	if !ok {
		t.Fatalf("expected a Model, got %T", model)
	}
	if moved.activeGroup != "group-01" {
		t.Fatalf("auto-select did not fire: activeGroup = %q", moved.activeGroup)
	}

	_, cmd := moved.Update(tea.KeyPressMsg{Code: 't', Text: "t"})

	var stopped bool
	for _, msg := range messagesFrom(cmd) {
		if dockerMsg, ok := msg.(cmds.RunDockerActionMsg); ok {
			if dockerMsg.Action == "stop" && dockerMsg.Target == "group-01" && dockerMsg.IsGroup {
				stopped = true
			}
		}
	}
	if !stopped {
		t.Errorf("t did not stop group-01, got %#v", messagesFrom(cmd))
	}
}

// Filtering is the one list feature that takes over the keyboard, and the panel
// verbs have to stand down while it does - otherwise typing "nginx" into the
// filter opens the new-group modal on the n.
func TestFilteringOwnsTheKeyboard(t *testing.T) {
	groups := focusedGroupsList(t, 12)

	if groups.OwnsKeyboard() {
		t.Fatal("a list with no filter should not own the keyboard")
	}

	model, _ := press(t, groups, "/")
	filtering, ok := model.(Model)
	if !ok {
		t.Fatalf("expected a Model, got %T", model)
	}
	if filtering.list.FilterState() != list.Filtering {
		t.Fatalf("/ did not start a filter, state is %v", filtering.list.FilterState())
	}
	if !filtering.OwnsKeyboard() {
		t.Error("a list being filtered should own the keyboard")
	}

	// n would create a group, d would delete one and space would select: typed
	// into a filter they are three letters of "nd ".
	model, msgs := press(t, filtering, "n")
	for _, msg := range msgs {
		switch msg.(type) {
		case cmds.OpenCreateGroupModalMsg, cmds.OpenDeleteGroupModalMsg, cmds.SetSelectedGroupMsg:
			t.Errorf("n acted as a command (%T) while the filter was being typed", msg)
		}
	}

	// The rest are driven without draining commands: the filter input returns a
	// cursor-blink command that costs half a second to run.
	for _, keystroke := range []string{"d", " "} {
		model, _ = model.Update(tea.KeyPressMsg{Code: rune(keystroke[0]), Text: keystroke})
	}

	typed, ok := model.(Model)
	if !ok {
		t.Fatalf("expected a Model, got %T", model)
	}
	if got := typed.list.FilterValue(); got != "nd " {
		t.Errorf("the filter holds %q, so the keystrokes did not all land as text", got)
	}
}

// Esc has to get out of a filter, which is the reason the list keeps esc at all
// while the app owns it everywhere else: without this there is no way back out
// of a filtered list.
func TestEscapeLeavesTheFilter(t *testing.T) {
	groups := focusedGroupsList(t, 12)

	model, _ := press(t, groups, "/")
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEscape})

	after, ok := model.(Model)
	if !ok {
		t.Fatalf("expected a Model, got %T", model)
	}
	if after.list.FilterState() != list.Unfiltered {
		t.Errorf("esc left the list in filter state %v", after.list.FilterState())
	}
	if after.OwnsKeyboard() {
		t.Error("the list still owns the keyboard after esc")
	}
}

// The keys the list keeps, because only the list can answer them.
func TestTheListKeepsItsOwnNavigation(t *testing.T) {
	groups := focusedGroupsList(t, 12)

	model, _ := press(t, groups, "G")
	atEnd, ok := model.(Model)
	if !ok {
		t.Fatalf("expected a Model, got %T", model)
	}
	if atEnd.list.Index() != len(atEnd.list.Items())-1 {
		t.Errorf("G left the cursor at index %d of %d", atEnd.list.Index(), len(atEnd.list.Items()))
	}

	model, _ = press(t, atEnd, "g")
	atStart, ok := model.(Model)
	if !ok {
		t.Fatalf("expected a Model, got %T", model)
	}
	if atStart.list.Index() != 0 {
		t.Errorf("g left the cursor at index %d rather than the first row", atStart.list.Index())
	}
}

// Auto-select: moving the cursor automatically selects the item under it.
func TestCursorMovementAutoSelects(t *testing.T) {
	groups := focusedGroupsList(t, 12)

	// Initial state: no selection.
	if groups.activeGroup != "" {
		t.Fatalf("initial activeGroup should be empty, got %q", groups.activeGroup)
	}

	// Move down one row (cursor goes from 0 to 1).
	model, cmd := groups.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	moved := model.(Model)

	// Auto-select should have fired (index 1 = group-01).
	if moved.activeGroup != "group-01" {
		t.Errorf("after j: activeGroup = %q, want group-01", moved.activeGroup)
	}

	// A SetSelectedGroupMsg should have been emitted.
	var selectedMsg bool
	for _, msg := range messagesFrom(cmd) {
		if _, ok := msg.(cmds.SetSelectedGroupMsg); ok {
			selectedMsg = true
		}
	}
	if !selectedMsg {
		t.Error("cursor movement did not emit SetSelectedGroupMsg")
	}

	// Move down again (cursor goes from 1 to 2).
	model, _ = moved.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	moved2 := model.(Model)

	if moved2.activeGroup != "group-02" {
		t.Errorf("after second j: activeGroup = %q, want group-02", moved2.activeGroup)
	}
}

// Moving up also auto-selects.
func TestCursorMovementUpAutoSelects(t *testing.T) {
	groups := focusedGroupsList(t, 12)

	// Move down twice (cursor goes to index 2).
	model, _ := groups.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	model, _ = model.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	moved := model.(Model)

	if moved.activeGroup != "group-02" {
		t.Fatalf("after two j: activeGroup = %q, want group-02", moved.activeGroup)
	}

	// Move up (cursor goes from 2 to 1).
	model, _ = moved.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	movedUp := model.(Model)

	if movedUp.activeGroup != "group-01" {
		t.Errorf("after k: activeGroup = %q, want group-01", movedUp.activeGroup)
	}
}
