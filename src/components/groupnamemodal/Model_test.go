package groupnamemodal

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/cmds"
)

// specialKey builds a KeyPressMsg for a special key (esc, enter) where
// Code alone resolves to the right string for key.Matches. Its own copy:
// ThemePickerModal_test.go's version was shared for free while both lived
// in the flat components package, and no longer is.
func specialKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

// renameModalFrame renders the rename modal as plain text, for assertions on
// its title, prefill and inline errors.
func renameModalFrame(m tea.Model) string {
	return ansi.Strip(m.View().Content)
}

// followRequest drains a CloseModalMsg's follow command and returns the
// request it produces, failing if the modal did not emit one.
func followRequest(t *testing.T, cmd tea.Cmd) *cmds.RenameGroupRequestMsg {
	t.Helper()

	msg := cmd()
	closeMsg, ok := msg.(cmds.CloseModalMsg)
	if !ok {
		t.Fatalf("expected CloseModalMsg, got %T", msg)
	}
	if closeMsg.Follow == nil {
		t.Fatal("modal closed without a follow command")
	}

	reqMsg := closeMsg.Follow()
	req, ok := reqMsg.(cmds.RenameGroupRequestMsg)
	if !ok {
		t.Fatalf("expected RenameGroupRequestMsg follow, got %T", reqMsg)
	}
	return &req
}

// The rename modal opens pre-filled with the current name and says what it
// is, so the user knows they are renaming rather than creating.
func TestRenameModalPrefillsTheCurrentName(t *testing.T) {
	m := NewForRename("core", nil)

	frame := renameModalFrame(m)
	if !strings.Contains(frame, "Rename group") {
		t.Errorf("modal title is not \"Rename group\":\n%s", frame)
	}
	if !strings.Contains(frame, "core") {
		t.Errorf("modal is not pre-filled with the current name:\n%s", frame)
	}
}

// Enter with the unchanged name is refused with its own message: it would
// otherwise rewrite the whole file for nothing.
func TestRenameModalRejectsTheUnchangedName(t *testing.T) {
	m := NewForRename("core", []string{"core", "extra"}).(Model)

	updated, cmd := m.Update(specialKey(tea.KeyEnter))
	m = updated.(Model)

	frame := renameModalFrame(m)
	if !strings.Contains(frame, "already named") {
		t.Errorf("unchanged name was not refused inline:\n%s", frame)
	}
	if cmd != nil {
		t.Error("an unchanged name produced a command; nothing should be written")
	}
}

// A new name that is another group's name is a collision, not a rename.
func TestRenameModalRejectsACollision(t *testing.T) {
	m := NewForRename("core", []string{"core", "edge"}).(Model)
	m.input.SetValue("edge")

	updated, cmd := m.Update(specialKey(tea.KeyEnter))
	m = updated.(Model)

	frame := renameModalFrame(m)
	if !strings.Contains(frame, "already exists") {
		t.Errorf("colliding name was not refused inline:\n%s", frame)
	}
	if cmd != nil {
		t.Error("a colliding name produced a command; nothing should be written")
	}
}

// The reserved ungrouped name cannot be claimed: cais shows a group of that
// name for every service with no profiles: key, so a real group by the same
// name would collide with it in the list and in every membership check. The
// refusal is its own message, not the generic "already exists" - the name is
// reserved even when no real group carries it.
func TestRenameModalRejectsTheReservedName(t *testing.T) {
	m := NewForRename("core", []string{"core", "extra"}).(Model)
	m.input.SetValue(apptypes.UngroupedGroup)

	updated, cmd := m.Update(specialKey(tea.KeyEnter))
	m = updated.(Model)

	frame := renameModalFrame(m)
	if !strings.Contains(frame, "reserved for services with no group") {
		t.Errorf("reserved name was not refused inline:\n%s", frame)
	}
	if cmd != nil {
		t.Error("a reserved name produced a command; nothing should be written")
	}
}

// The create flow goes through the same submit handler, so the reserved name
// is refused there too.
func TestCreateModalRejectsTheReservedName(t *testing.T) {
	m := New([]string{"core"}, []string{"web", "db"}, 24).(Model)
	m.input.SetValue(apptypes.UngroupedGroup)

	updated, cmd := m.Update(specialKey(tea.KeyEnter))
	m = updated.(Model)

	frame := renameModalFrame(m)
	if !strings.Contains(frame, "reserved for services with no group") {
		t.Errorf("reserved name was not refused inline:\n%s", frame)
	}
	if cmd != nil {
		t.Error("a reserved name produced a command; nothing should be written")
	}
}

// A valid rename closes the modal with a rename request carrying both names.
func TestRenameModalEmitsARenameRequest(t *testing.T) {
	m := NewForRename("core", []string{"core", "extra"}).(Model)
	m.input.SetValue("core2")

	_, cmd := m.Update(specialKey(tea.KeyEnter))

	req := followRequest(t, cmd)
	if req.GroupName != "core" {
		t.Errorf("request group = %q, want %q", req.GroupName, "core")
	}
	if req.NewName != "core2" {
		t.Errorf("request new name = %q, want %q", req.NewName, "core2")
	}
}

// Esc cancels the rename without emitting anything.
func TestRenameModalEscClosesWithoutRequest(t *testing.T) {
	m := NewForRename("core", nil)

	_, cmd := m.Update(specialKey(tea.KeyEsc))

	msg := cmd()
	if _, ok := msg.(cmds.CloseModalMsg); !ok {
		t.Fatalf("expected CloseModalMsg on Esc, got %T", msg)
	}
	if msg.(cmds.CloseModalMsg).Follow != nil {
		t.Error("Esc must not emit a rename request")
	}
}
