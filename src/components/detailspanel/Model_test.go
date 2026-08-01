package detailspanel

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

func focusedDetails(service types.ServiceConfig) Model {
	m := New(&service).(Model)
	m = m.applySize().(Model)
	m.isFocused = true
	return m
}

func (m Model) applySize() tea.Model {
	updated, _ := m.Update(cmds.SetBodyLayoutMsg{LeftWidth: 40, RightWidth: 60, Height: 24})
	return updated
}

// collectMessages drains a command, returning every message it produces.
// tea.Batch wraps its children in a BatchMsg, so a single Update can yield
// several.
func collectMessages(t *testing.T, cmd tea.Cmd) []tea.Msg {
	t.Helper()

	if cmd == nil {
		return nil
	}

	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, child := range batch {
			msgs = append(msgs, collectMessages(t, child)...)
		}
		return msgs
	}

	return []tea.Msg{msg}
}

func findMessageOfType[T any](t *testing.T, cmd tea.Cmd) (T, bool) {
	t.Helper()

	var zero T
	for _, msg := range collectMessages(t, cmd) {
		if m, ok := msg.(T); ok {
			return m, true
		}
	}

	return zero, false
}

func hasMessageOfType[T any](t *testing.T, cmd tea.Cmd) bool {
	t.Helper()

	_, ok := findMessageOfType[T](t, cmd)
	return ok
}

func TestDetailsPanelEEntersEditMode(t *testing.T) {
	m := focusedDetails(types.ServiceConfig{Name: "web"})

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
	m = updated.(Model)

	if !hasMessageOfType[cmds.RequestInlineEditMsg](t, cmd) {
		t.Fatalf("expected RequestInlineEditMsg, got %T", collectMessages(t, cmd))
	}

	if m.editing {
		t.Fatal("panel should not enter edit mode until the fragment arrives")
	}

	updated, cmd = m.Update(cmds.InlineEditReadyMsg{ServiceName: "web", Fragment: []byte("web:\n  image: nginx\n")})
	m = updated.(Model)

	if !m.editing {
		t.Fatal("panel should be in edit mode after receiving the fragment")
	}

	if !strings.Contains(m.editor.Value(), "web:") {
		t.Fatalf("editor did not receive the fragment: %q", m.editor.Value())
	}

	if !hasMessageOfType[cmds.SetEditingStateMsg](t, cmd) {
		t.Fatalf("expected SetEditingState command, got %T", collectMessages(t, cmd))
	}
}

func TestDetailsPanelEditModeSwallowsActionKeys(t *testing.T) {
	m := focusedDetails(types.ServiceConfig{Name: "web"})
	m, _ = m.enterEditMode([]byte("web:\n  image: nginx\n"))
	m.isFocused = true

	for _, letter := range []rune{'s', 't', 'r', 'p', 'x', 'l'} {
		updated, _ := m.Update(tea.KeyPressMsg{Code: letter, Text: string(letter)})
		m = updated.(Model)

		if !strings.Contains(m.editor.Value(), string(letter)) {
			t.Errorf("%c in edit mode did not reach the textarea; value = %q", letter, m.editor.Value())
		}
	}

	if m.OwnsKeyboard() != true {
		t.Fatal("editing panel should own the keyboard")
	}
}

func TestDetailsPanelCtrlSSaves(t *testing.T) {
	m := focusedDetails(types.ServiceConfig{Name: "web"})
	m, _ = m.enterEditMode([]byte("web:\n  image: nginx\n"))
	m.isFocused = true

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 's', Text: "ctrl+s", Mod: tea.ModCtrl})
	m = updated.(Model)

	if !hasMessageOfType[cmds.RequestSaveServiceMsg](t, cmd) {
		t.Fatalf("expected RequestSaveServiceMsg, got %T", collectMessages(t, cmd))
	}

	if m.saveError != "" {
		t.Fatalf("saveError should be empty before save response, got %q", m.saveError)
	}
}

func TestDetailsPanelEscCancelsWithoutChanges(t *testing.T) {
	m := focusedDetails(types.ServiceConfig{Name: "web"})
	m, _ = m.enterEditMode([]byte("web:\n  image: nginx\n"))
	m.isFocused = true

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc, Text: "esc"})
	m = updated.(Model)

	if m.editing {
		t.Fatal("esc should exit edit mode when there are no changes")
	}

	if !hasMessageOfType[cmds.SetEditingStateMsg](t, cmd) {
		t.Fatalf("expected SetEditingState command, got %T", collectMessages(t, cmd))
	}
}

func TestDetailsPanelEscWithChangesOpensConfirm(t *testing.T) {
	m := focusedDetails(types.ServiceConfig{Name: "web"})
	m, _ = m.enterEditMode([]byte("web:\n  image: nginx\n"))
	m.isFocused = true

	// Type an extra space into the editor so the value differs.
	m.editor.SetValue(m.editor.Value() + " ")

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc, Text: "esc"})
	m = updated.(Model)

	if !m.editing {
		t.Fatal("esc should not immediately exit edit mode when there are changes")
	}

	if !hasMessageOfType[cmds.OpenConfirmModalMsg](t, cmd) {
		t.Fatalf("expected OpenConfirmModalMsg, got %T", collectMessages(t, cmd))
	}
}

func TestDetailsPanelCtrlOOpensExternalEditor(t *testing.T) {
	m := focusedDetails(types.ServiceConfig{Name: "web"})
	m, _ = m.enterEditMode([]byte("web:\n  image: nginx\n"))
	m.isFocused = true

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'o', Text: "ctrl+o", Mod: tea.ModCtrl})
	m = updated.(Model)

	if !hasMessageOfType[cmds.OpenServiceEditorMsg](t, cmd) {
		t.Fatalf("expected OpenServiceEditorMsg, got %T", collectMessages(t, cmd))
	}

	if !m.editing {
		t.Fatal("ctrl+o should keep the editor open while the external editor runs")
	}
}

func TestDetailsPanelServiceSavedSuccessExitsEditMode(t *testing.T) {
	m := focusedDetails(types.ServiceConfig{Name: "web"})
	m, _ = m.enterEditMode([]byte("web:\n  image: nginx\n"))

	updated, cmd := m.Update(cmds.ServiceSavedMsg{ServiceName: "web"})
	m = updated.(Model)

	if m.editing {
		t.Fatal("successful save should exit edit mode")
	}

	if !hasMessageOfType[cmds.SetEditingStateMsg](t, cmd) {
		t.Fatalf("expected SetEditingState command, got %T", collectMessages(t, cmd))
	}
}

func TestDetailsPanelServiceSavedFailureShowsInlineError(t *testing.T) {
	m := focusedDetails(types.ServiceConfig{Name: "web"})
	m, _ = m.enterEditMode([]byte("web:\n  image: nginx\n"))

	updated, _ := m.Update(cmds.ServiceSavedMsg{ServiceName: "web", Err: errBoom{}})
	m = updated.(Model)

	if !m.editing {
		t.Fatal("failed save should keep the editor open")
	}

	if m.saveError == "" {
		t.Fatal("failed save should set saveError")
	}
}

func TestDetailsPanelLiveValidationReportsBadYAML(t *testing.T) {
	m := focusedDetails(types.ServiceConfig{Name: "web"})
	m, _ = m.enterEditMode([]byte("web:\n  image: nginx\n"))

	m.editor.SetValue("web: : : bad")
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	m = updated.(Model)

	if !strings.Contains(m.editor.Value(), "bad") {
		t.Fatalf("editor value not set: %q", m.editor.Value())
	}

	if m.validationError == "" {
		t.Fatal("bad YAML should set validationError")
	}
}

// TestEditorHintsNameTheEditorKeys asserts the editor hint line reads key
// names from their bindings, not from string literals.
func TestEditorHintsNameTheEditorKeys(t *testing.T) {
	m := focusedDetails(types.ServiceConfig{Name: "web"})
	m, _ = m.enterEditMode([]byte("web:\n  image: nginx\n"))

	rendered := m.renderEditorHints(80)

	for _, want := range []string{
		keys.Details.Save.Help().Key,
		keys.Details.OpenEditor.Help().Key,
		keys.Editor.Indent.Help().Key,
		keys.Editor.Outdent.Help().Key,
		keys.Global.Back.Help().Key,
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("editor hints do not mention %q\n  rendered: %q", want, rendered)
		}
	}
}

type errBoom struct{}

func (errBoom) Error() string { return "boom" }
