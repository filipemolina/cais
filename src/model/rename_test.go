package model

import (
	"testing"

	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components/groupnamemodal"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"strings"
)

// R on the groups list opens the rename modal for the highlighted group.
// The modal is the rename one: it says "Rename group", not "New group".
func TestPressingROpensTheRenameModal(t *testing.T) {
	m := withGroupsLoaded(t)

	// The list cursor is on the first row, which is "core" after the sorted
	// sync - the same setup TestPressingEOpensTheMembershipEditor relies on.
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'R', Text: "R"})
	m = drive(updated, collect(cmd)...)

	if m.activeModal == nil {
		t.Fatal("R did not open a modal")
	}

	if _, ok := m.activeModal.(groupnamemodal.Model); !ok {
		t.Fatalf("R opened %T, want a groupnamemodal.Model", m.activeModal)
	}

	frame := ansi.Strip(m.activeModal.View().Content)
	if !strings.Contains(frame, "Rename group") {
		t.Errorf("R opened the create prompt, not the rename prompt:\n%s", frame)
	}
}

// Typing a new name and pressing Enter drives the whole rename: the modal
// closes and its request reaches AppModel, which turns it into a RenameGroup
// command carrying the loaded file - the same two-step as the membership
// editor.
func TestRenameFlowsFromModalToLoadedFile(t *testing.T) {
	m := withGroupsLoaded(t)

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'R', Text: "R"})
	m = drive(updated, collect(cmd)...)

	// The input is pre-filled with "core"; typing appends at the cursor.
	// The textinput returns its own cursor command; what matters is that the
	// modal stays open.
	updated, _ = m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	m = updated.(AppModel)
	if m.activeModal == nil {
		t.Fatal("typing into the modal closed it")
	}

	// Enter submits. The modal closes with the rename request as its
	// follow-up command, which surfaces only once AppModel has processed
	// the close.
	updated, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(AppModel)

	var gotReq *cmds.RenameGroupRequestMsg
	var closed bool
	for _, msg := range collect(cmd) {
		if _, ok := msg.(cmds.CloseModalMsg); ok {
			closed = true
			updated, closeCmd := m.Update(msg)
			m = updated.(AppModel)
			for _, follow := range collect(closeCmd) {
				if req, ok := follow.(cmds.RenameGroupRequestMsg); ok {
					r := req
					gotReq = &r
				}
			}
		}
	}
	if !closed {
		t.Fatal("Enter did not close the rename modal")
	}
	if gotReq == nil {
		t.Fatal("Enter did not emit a RenameGroupRequestMsg")
	}
	if gotReq.GroupName != "core" {
		t.Errorf("request group = %q, want %q", gotReq.GroupName, "core")
	}
	if gotReq.NewName != "core2" {
		t.Errorf("request new name = %q, want %q", gotReq.NewName, "core2")
	}

	// AppModel answers with a RenameGroup command bound to the loaded file.
	_, cmd = m.Update(*gotReq)
	var gotRename bool
	for _, msg := range collect(cmd) {
		if _, ok := msg.(cmds.RenameGroupMsg); ok {
			gotRename = true
		}
	}
	if !gotRename {
		t.Error("RenameGroupRequestMsg did not produce a RenameGroupMsg")
	}
}

// A successful rename reloads the config and keeps the renamed group
// selected: selection.groupName still holds the old name, so the handler
// moves it to the new one before the reload re-syncs the lists.
func TestRenameSuccessReloadsAndKeepsSelection(t *testing.T) {
	m := withGroupsLoaded(t)

	if m.selection.groupName != "core" {
		t.Fatalf("precondition: selected group = %q, want %q", m.selection.groupName, "core")
	}

	updated, cmd := m.Update(cmds.RenameGroupMsg{Err: nil, NewName: "core2"})
	m = updated.(AppModel)

	if m.selection.groupName != "core2" {
		t.Errorf("selection after rename = %q, want %q", m.selection.groupName, "core2")
	}

	var reloaded bool
	for _, msg := range collect(cmd) {
		if _, ok := msg.(cmds.GetConfigMsg); ok {
			reloaded = true
		}
	}
	if !reloaded {
		t.Error("a successful rename did not trigger a config reload")
	}
}

// A rename failure surfaces in the error banner and leaves the old
// selection standing.
func TestRenameFailureShowsErrorAndKeepsSelection(t *testing.T) {
	m := withGroupsLoaded(t)

	updated, _ := m.Update(cmds.RenameGroupMsg{Err: errBoom{}, NewName: "core2"})
	m = updated.(AppModel)

	if m.lastError == "" {
		t.Error("a failed rename left no error")
	}
	if m.selection.groupName != "core" {
		t.Errorf("selection after a failed rename = %q, want %q (unchanged)", m.selection.groupName, "core")
	}
}
