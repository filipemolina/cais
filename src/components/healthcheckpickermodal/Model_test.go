package healthcheckpickermodal

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/stack-stitcher/src/cmds"
)

func specialKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

func letterKey(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

func frame(m tea.Model) string {
	return ansi.Strip(m.View().Content)
}

// followRequest drains a CloseModalMsg's follow command and returns the
// message it produces.
func followRequest(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()

	if cmd == nil {
		t.Fatal("expected a command, got nil")
	}
	msg := cmd()
	closeMsg, ok := msg.(cmds.CloseModalMsg)
	if !ok {
		t.Fatalf("expected CloseModalMsg, got %T", msg)
	}
	if closeMsg.Follow == nil {
		t.Fatal("modal closed without a follow command")
	}

	return closeMsg.Follow()
}

// A recognised image lists its template first; the generic fallback is
// never filtered out, only ever sorted last.
func TestPickerOrdersImageMatchFirst(t *testing.T) {
	m := New("db", types.ServiceConfig{Image: "postgres:16"}, 40)

	out := frame(m)
	if strings.Index(out, "PostgreSQL") > strings.Index(out, "Generic HTTP") {
		t.Errorf("PostgreSQL should list before Generic HTTP, got:\n%s", out)
	}
}

// The port field is hidden until the generic template is highlighted, and
// prefilled from the service's first published port's target.
func TestPortFieldVisibilityFollowsSelection(t *testing.T) {
	svc := types.ServiceConfig{
		Image: "myapp:latest", // matches nothing - Generic HTTP is the only row
		Ports: []types.ServicePortConfig{{Published: "18096", Target: 8096}},
	}
	m := New("web", svc, 40)

	if !strings.Contains(frame(m), "Port inside the container") {
		t.Errorf("generic template is the only row, so the port field should already be visible:\n%s", frame(m))
	}
	if !strings.Contains(frame(m), "8096") {
		t.Errorf("port field should prefill from the first published port's target, got:\n%s", frame(m))
	}
}

func TestPortFieldHiddenForAnImageMatchedTemplate(t *testing.T) {
	m := New("db", types.ServiceConfig{Image: "postgres:16"}, 40)

	if strings.Contains(frame(m), "Port inside the container") {
		t.Errorf("PostgreSQL is highlighted first and needs no port field:\n%s", frame(m))
	}
}

// Enter on a non-generic template submits immediately with no port.
func TestSubmitNonGenericTemplate(t *testing.T) {
	m := New("db", types.ServiceConfig{Image: "postgres:16"}, 40)

	updated, cmd := m.Update(specialKey(tea.KeyEnter))
	_ = updated

	msg := followRequest(t, cmd)
	req, ok := msg.(cmds.AddHealthcheckRequestMsg)
	if !ok {
		t.Fatalf("expected AddHealthcheckRequestMsg, got %T", msg)
	}
	if req.ServiceName != "db" || req.Template.Name != "PostgreSQL" || req.Port != "" {
		t.Errorf("got %+v, want ServiceName=db Template=PostgreSQL Port empty", req)
	}
}

// Submitting the generic template with an empty port is refused, and does
// not close the modal.
func TestSubmitGenericTemplateRequiresAPort(t *testing.T) {
	svc := types.ServiceConfig{Image: "myapp:latest"}
	m := New("web", svc, 40)

	// Clear the prefilled port.
	for range 10 {
		m, _ = m.Update(specialKey(tea.KeyBackspace))
	}

	updated, cmd := m.Update(specialKey(tea.KeyEnter))

	if cmd != nil {
		if msg := cmd(); msg != nil {
			if _, ok := msg.(cmds.CloseModalMsg); ok {
				t.Fatal("modal closed with an empty port")
			}
		}
	}
	if !strings.Contains(frame(updated), "empty") {
		t.Errorf("expected an inline error about the empty port, got:\n%s", frame(updated))
	}
}

// Typing into the port field only happens while the generic template is
// selected - this is the visibility-derives-behaviour rule the plan
// describes, checked by typing a digit and confirming it lands in the
// field rather than being swallowed by the list.
func TestTypingReachesThePortFieldWhenGenericIsSelected(t *testing.T) {
	svc := types.ServiceConfig{Image: "myapp:latest"}
	m := New("web", svc, 40)

	for range 10 {
		m, _ = m.Update(specialKey(tea.KeyBackspace))
	}
	m, _ = m.Update(letterKey('9'))

	mm := m.(Model)
	if mm.portInput.Value() != "9" {
		t.Errorf("port field = %q, want %q", mm.portInput.Value(), "9")
	}
}

func TestCancelClosesWithoutARequest(t *testing.T) {
	m := New("db", types.ServiceConfig{Image: "postgres:16"}, 40)

	_, cmd := m.Update(specialKey(tea.KeyEsc))
	if cmd == nil {
		t.Fatal("esc produced no command")
	}
	msg := cmd()
	closeMsg, ok := msg.(cmds.CloseModalMsg)
	if !ok {
		t.Fatalf("expected CloseModalMsg, got %T", msg)
	}
	if closeMsg.Follow != nil {
		t.Error("cancel should not carry a follow command")
	}
}

// Cursor movement is handled directly (list.CursorUp/CursorDown) rather
// than by forwarding the keypress to list.Update, specifically so the
// bubbles default keymap (which claims h/l/b/u/f/d/g/G/q, among others)
// never intercepts a key that should have gone to the port field. This
// pins that down keeps the selection working across the sub-components.
func TestArrowKeysMoveTheSelection(t *testing.T) {
	m := New("db", types.ServiceConfig{Image: "postgres:16"}, 40).(Model)

	first, _ := m.selectedTemplate()
	if first.Name != "PostgreSQL" {
		t.Fatalf("precondition: expected PostgreSQL first, got %q", first.Name)
	}

	updated, _ := m.Update(specialKey(tea.KeyDown))
	mm := updated.(Model)
	second, ok := mm.selectedTemplate()
	if !ok || second.Name == first.Name {
		t.Errorf("down did not move the selection, still on %q", second.Name)
	}
}
