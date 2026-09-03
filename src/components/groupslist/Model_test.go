package groupslist

import (
	"fmt"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/cmds"
)

// groupNames makes n distinct group names, enough of them to force the list
// onto more than one page.
func groupNames(n int) cmds.SetGroupsListMsg {
	names := make([]cmds.GroupStatus, 0, n)
	for i := range n {
		names = append(names, cmds.GroupStatus{Name: fmt.Sprintf("group-%02d", i)})
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

// ungroupedSelectedList is a focused groups list whose cursor sits on the
// reserved ungrouped row.
func ungroupedSelectedList(t *testing.T) Model {
	t.Helper()

	var model tea.Model = New(nil, 40, 20)
	for _, msg := range []tea.Msg{
		cmds.SetBodyLayoutMsg{LeftWidth: 40, Height: 20},
		cmds.SetGroupsListMsg([]cmds.GroupStatus{{Name: "core"}, {Name: apptypes.UngroupedGroup}}),
	} {
		model, _ = model.Update(msg)
	}

	// Move the cursor down to the ungrouped row; auto-select fires on cursor
	// movement, the same way it does in the real app.
	model, _ = model.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})

	groups, ok := model.(Model)
	if !ok {
		t.Fatalf("expected a Model, got %T", model)
	}
	if groups.activeGroup != apptypes.UngroupedGroup {
		t.Fatalf("precondition: activeGroup = %q, want %q", groups.activeGroup, apptypes.UngroupedGroup)
	}

	return groups
}

// The reserved ungrouped row is read-only: d and e refuse it - there is
// no profile tag behind it to delete or reconcile membership against. (Rename
// was folded into e, so e already covers the edit path.)
func TestListManagementKeysRefuseTheUngroupedRow(t *testing.T) {
	for _, keystroke := range []string{"d", "e"} {
		t.Run(keystroke, func(t *testing.T) {
			groups := ungroupedSelectedList(t)

			_, msgs := press(t, groups, keystroke)

			for _, msg := range msgs {
				switch msg.(type) {
				case cmds.OpenDeleteGroupModalMsg, cmds.OpenEditGroupModalMsg, cmds.OpenRenameGroupModalMsg:
					t.Errorf("%s opened a modal on the ungrouped row: %#v", keystroke, msgs)
				}
			}
		})
	}
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

// A on the reserved ungrouped row asks AppModel to show the adopt/release
// confirm. The list does not know whether the row is derived or materialized -
// that is a fact about the loaded project, so it sends a generic request.
func TestUngroupedKeyRequestsTheToggleOnTheUngroupedRow(t *testing.T) {
	groups := ungroupedSelectedList(t)

	_, msgs := press(t, groups, "A")

	var requested bool
	for _, msg := range msgs {
		if _, ok := msg.(cmds.ToggleUngroupedRequestMsg); ok {
			requested = true
		}
	}
	if !requested {
		t.Errorf("A did not request the ungrouped toggle on the ungrouped row, got %#v", msgs)
	}
}

// A on a real group does nothing: the toggle is the reserved row's verb alone.
func TestUngroupedKeyRefusesRealGroups(t *testing.T) {
	groups := focusedGroupsList(t, 12)

	_, msgs := press(t, groups, "A")

	for _, msg := range msgs {
		if _, ok := msg.(cmds.ToggleUngroupedRequestMsg); ok {
			t.Errorf("A requested the ungrouped toggle on a real group: %#v", msgs)
		}
	}
}

// The list keeps esc only while it can use it: with a filter standing. Both
// body panels are always active now, so focus is no longer part of the
// condition - only the filter state matters. Mid-typing it owns the whole
// keyboard instead.
func TestKeepsEscOnlyWhileAnAppliedFilterStands(t *testing.T) {
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
		t.Error("a list with a filter standing did not keep esc")
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
	want := "1 group · 3 services · 1 running · 2 ungrouped"
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

	full := "1 group · 3 services · 1 running · 2 ungrouped"
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

// settle feeds messages through the list and then keeps feeding back whatever
// the resulting commands produced, the way the Bubble Tea runtime does. It is
// the only way to assert on behaviour under a filter: the filtered rows are
// populated by list.FilterMatchesMsg, which arrives as a command. Selections
// are pulled out of the loop and returned rather than fed back, so a test can
// see what the list asked AppModel to select.
func settle(t *testing.T, model Model, msgs ...tea.Msg) (Model, []string) {
	t.Helper()

	var selected []string

	queue := append([]tea.Msg{}, msgs...)
	for range 200 {
		if len(queue) == 0 {
			break
		}

		msg := queue[0]
		queue = queue[1:]

		next, cmd := model.Update(msg)
		updated, ok := next.(Model)
		if !ok {
			t.Fatalf("expected a Model, got %T", next)
		}
		model = updated

		for _, produced := range messagesFrom(cmd) {
			if produced == nil {
				continue
			}
			if pick, ok := produced.(cmds.SetSelectedGroupMsg); ok {
				selected = append(selected, string(pick))
				continue
			}
			queue = append(queue, produced)
		}
	}

	return model, selected
}

// visibleNames is what the list is actually showing, in order.
func visibleNames(m Model) []string {
	names := make([]string, 0, len(m.list.VisibleItems()))
	for _, item := range m.list.VisibleItems() {
		if group, ok := item.(apptypes.GroupListItem); ok {
			names = append(names, group.Name)
		}
	}

	return names
}

// filteredGroups is a groups list narrowed to the groups matching term.
func filteredGroups(t *testing.T, term string, names ...string) (Model, []string) {
	t.Helper()

	statuses := make([]cmds.GroupStatus, 0, len(names))
	for _, name := range names {
		statuses = append(statuses, cmds.GroupStatus{Name: name})
	}

	var model tea.Model = New(nil, 40, 20)
	for _, msg := range []tea.Msg{
		cmds.SetBodyLayoutMsg{LeftWidth: 40, Height: 20},
		cmds.SetGroupsListMsg(statuses),
		cmds.SetSelectedGroupMsg(names[0]),
	} {
		model, _ = model.Update(msg)
	}

	groups, ok := model.(Model)
	if !ok {
		t.Fatalf("expected a Model, got %T", model)
	}

	msgs := []tea.Msg{tea.KeyPressMsg{Code: '/', Text: "/"}}
	for _, r := range term {
		msgs = append(msgs, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	msgs = append(msgs, tea.KeyPressMsg{Code: tea.KeyEnter})

	return settle(t, groups, msgs...)
}

// Regression test: applying a filter while the cursor is already on the first
// row left the group details panel showing a group the filter had just
// hidden. The selection used to be published only when list.Index() changed,
// and accepting a filter sends the cursor to the top - so a cursor already at
// the top kept index 0 while row 0 became a different group entirely.
func TestFilterSelectsTheGroupItPutsUnderTheCursor(t *testing.T) {
	filtered, selected := filteredGroups(t, "media", "core", "media", "tools")

	if got := visibleNames(filtered); len(got) == 0 || got[0] != "media" {
		t.Fatalf("precondition: filtered rows are %v, want media first", got)
	}
	if got := filtered.activeGroup; got != "media" {
		t.Errorf("activeGroup = %q, want %q", got, "media")
	}
	if len(selected) == 0 {
		t.Fatal("no SetSelectedGroupMsg was published; the details panel keeps the pre-filter group")
	}
	if got := selected[len(selected)-1]; got != "media" {
		t.Errorf("last published selection = %q, want %q", got, "media")
	}
}

// Regression test: the delegate's activeIndex is an index into the rows being
// drawn, which list.populatedView numbers against VisibleItems. Deriving it
// from the full Items slice put the highlight on the wrong row under a
// filter, or past the end of the visible rows and so on no row at all.
func TestGroupActiveIndexIsRelativeToTheVisibleRows(t *testing.T) {
	filtered, _ := filteredGroups(t, "media", "core", "media", "tools")

	visible := visibleNames(filtered)
	if len(visible) == 0 {
		t.Fatal("precondition: the filter hid everything")
	}

	active := filtered.listDelegate.activeIndex
	if active < 0 || active >= len(visible) {
		t.Fatalf("activeIndex = %d, outside the %d visible rows %v",
			active, len(visible), visible)
	}
	if got := visible[active]; got != filtered.activeGroup {
		t.Errorf("activeIndex %d points at %q, but the selected group is %q",
			active, got, filtered.activeGroup)
	}
}

// A selection set by AppModel must not be overridden by the row the cursor
// happens to be resting on. The cursor did not move, so the list has nothing
// new to report.
func TestExternalGroupSelectionIsNotEchoedBack(t *testing.T) {
	var model tea.Model = New(nil, 40, 20)
	for _, msg := range []tea.Msg{
		cmds.SetBodyLayoutMsg{LeftWidth: 40, Height: 20},
		cmds.SetGroupsListMsg([]cmds.GroupStatus{{Name: "core"}, {Name: "media"}, {Name: "tools"}}),
	} {
		model, _ = model.Update(msg)
	}

	groups, ok := model.(Model)
	if !ok {
		t.Fatalf("expected a Model, got %T", model)
	}

	updated, selected := settle(t, groups, cmds.SetSelectedGroupMsg("tools"))

	if got := updated.activeGroup; got != "tools" {
		t.Errorf("activeGroup = %q, want tools", got)
	}
	if len(selected) != 0 {
		t.Errorf("published %d selections, want none: AppModel's choice was echoed back", len(selected))
	}
}
