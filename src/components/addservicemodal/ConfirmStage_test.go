package addservicemodal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/filipemolina/stack-stitcher/src/cmds"
)

// toConfirm drives the search stage to the confirm stage with query typed
// and no results fed in, so Enter falls through to the typed text (D3).
func toConfirm(t *testing.T, m Model, query string) Model {
	t.Helper()

	m = typeInto(m, query)
	updated, _ := m.Update(specialKey(tea.KeyEnter))
	m = updated.(Model)

	if m.step != stepConfirm {
		t.Fatalf("failed to reach stepConfirm for %q (step = %v)", query, m.step)
	}
	return m
}

func TestConfirmStageValidationMatchesServicefieldsstep(t *testing.T) {
	cases := []struct {
		name  string
		image string
		want  string
	}{
		{"", "nginx:alpine", "Service name can't be empty"},
		{"web", "", "Image can't be empty (e.g. nginx:alpine)"},
		{"web service", "nginx:alpine", `"web service" is not a valid service name`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := toConfirm(t, New("compose.yaml", []string{"db"}).(Model), "sonarr")
			m.serviceName.SetValue(c.name)
			m.image.SetValue(c.image)

			updated, cmd := m.Update(specialKey(tea.KeyEnter))
			m = updated.(Model)

			if cmd != nil {
				t.Errorf("an invalid submit produced a command: %v", cmd())
			}
			if m.confirmErr != c.want {
				t.Errorf("confirmErr = %q, want %q", m.confirmErr, c.want)
			}
		})
	}
}

func TestConfirmStageFlagsADuplicateNameOnRender(t *testing.T) {
	m := New("compose.yaml", []string{"sonarr", "db"}).(Model)
	m = typeInto(m, "sonarr") // derives to the colliding name

	updated, _ := m.Update(specialKey(tea.KeyEnter)) // advance to confirm
	m = updated.(Model)

	if m.confirmErr == "" {
		t.Fatal("a duplicate name was not flagged the moment the confirm stage rendered")
	}
	if !strings.Contains(m.confirmErr, "sonarr") || !strings.Contains(m.confirmErr, "already exists") {
		t.Errorf("confirmErr = %q, want it to name the collision", m.confirmErr)
	}
	// The error is visible in the rendered view before any key is pressed.
	if !strings.Contains(ansi.Strip(m.View().Content), "already exists") {
		t.Error("the collision message is not visible in View() on render")
	}
}

func TestConfirmStageTabMovesFocusBothDirections(t *testing.T) {
	m := toConfirm(t, New("compose.yaml", []string{"db"}).(Model), "sonarr")

	if !m.serviceName.Focused() {
		t.Fatal("serviceName should be focused when the confirm stage renders")
	}

	updated, _ := m.Update(specialKey(tea.KeyTab))
	m = updated.(Model)
	if !m.image.Focused() {
		t.Error("Tab did not move focus to the image field")
	}

	updated, _ = m.Update(specialKey(tea.KeyTab))
	m = updated.(Model)
	if !m.serviceName.Focused() {
		t.Error("Tab did not move focus back to the service name field")
	}
}

func TestConfirmStageEscClosesTheWholeModal(t *testing.T) {
	m := toConfirm(t, New("compose.yaml", []string{"db"}).(Model), "sonarr")

	_, cmd := m.Update(specialKey(tea.KeyEsc))

	msg := cmd()
	closeMsg, ok := msg.(cmds.CloseModalMsg)
	if !ok {
		t.Fatalf("expected CloseModalMsg, got %T", msg)
	}
	if closeMsg.Follow != nil {
		t.Error("Esc should close the modal with no follow command")
	}
}

func TestConfirmStageSubmitDispatchesAddService(t *testing.T) {
	path := filepath.Join(t.TempDir(), "compose.yaml")
	if err := os.WriteFile(path, []byte("services:\n  web:\n    image: nginx:alpine\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := toConfirm(t, New(path, []string{"web"}).(Model), "sonarr")

	_, cmd := m.Update(specialKey(tea.KeyEnter))

	msg := followRequest(t, cmd)
	addMsg, ok := msg.(cmds.AddServiceMsg)
	if !ok {
		t.Fatalf("expected AddServiceMsg from AddService's cmd, got %T", msg)
	}
	if addMsg.Err != nil {
		t.Fatalf("AddService: %v", addMsg.Err)
	}
	if addMsg.ServiceName != "sonarr" || addMsg.Image != "sonarr" {
		t.Errorf("got ServiceName=%q Image=%q, want sonarr/sonarr", addMsg.ServiceName, addMsg.Image)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "sonarr:") {
		t.Errorf("the new service was not written, got:\n%s", contents)
	}
}
