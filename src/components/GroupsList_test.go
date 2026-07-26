package components

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/cmds"
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
func focusedGroupsList(t *testing.T, n int) GroupListModel {
	t.Helper()

	var model tea.Model = GroupsList(nil, 40, 20)
	for _, msg := range []tea.Msg{
		cmds.SetBodyLayoutMsg{LeftWidth: 40, Height: 20},
		groupNames(n),
		cmds.SetFocusMsg(1),
	} {
		model, _ = model.Update(msg)
	}

	groups, ok := model.(GroupListModel)
	if !ok {
		t.Fatalf("expected a GroupListModel, got %T", model)
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

	after, ok := model.(GroupListModel)
	if !ok {
		t.Fatalf("expected a GroupListModel, got %T", model)
	}
	if after.list.Paginator.Page != startPage {
		t.Errorf("d moved the list from page %d to page %d", startPage, after.list.Paginator.Page)
	}
}

// The other letters the default map claimed. None of them is a list key in this
// app: l opens logs, f follows a log stream, and h, b and u mean nothing yet,
// which is still not "page the list".
func TestPanelLettersDoNotPageTheList(t *testing.T) {
	for _, keystroke := range []string{"l", "h", "f", "b", "u"} {
		t.Run(keystroke, func(t *testing.T) {
			groups := focusedGroupsList(t, 12)
			startPage := groups.list.Paginator.Page

			model, _ := press(t, groups, keystroke)

			after, ok := model.(GroupListModel)
			if !ok {
				t.Fatalf("expected a GroupListModel, got %T", model)
			}
			if after.list.Paginator.Page != startPage {
				t.Errorf("%q moved the list from page %d to page %d", keystroke, startPage, after.list.Paginator.Page)
			}
		})
	}
}

// The keys the list keeps, because only the list can answer them.
func TestTheListKeepsItsOwnNavigation(t *testing.T) {
	groups := focusedGroupsList(t, 12)

	model, _ := press(t, groups, "G")
	atEnd, ok := model.(GroupListModel)
	if !ok {
		t.Fatalf("expected a GroupListModel, got %T", model)
	}
	if atEnd.list.Index() != len(atEnd.list.Items())-1 {
		t.Errorf("G left the cursor at index %d of %d", atEnd.list.Index(), len(atEnd.list.Items()))
	}

	model, _ = press(t, atEnd, "g")
	atStart, ok := model.(GroupListModel)
	if !ok {
		t.Fatalf("expected a GroupListModel, got %T", model)
	}
	if atStart.list.Index() != 0 {
		t.Errorf("g left the cursor at index %d rather than the first row", atStart.list.Index())
	}
}
