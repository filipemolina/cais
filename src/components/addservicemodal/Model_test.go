package addservicemodal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/cmds"
)

func specialKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

func letterKey(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

// followRequest drains a CloseModalMsg's follow command and returns the
// message it produces, failing if the modal did not close with one.
func followRequest(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()

	if cmd == nil {
		t.Fatal("submit produced no command")
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

func typeAndSubmit(t *testing.T, m tea.Model, name, image string) tea.Msg {
	t.Helper()

	var cmd tea.Cmd
	for _, ch := range name {
		m, cmd = m.Update(letterKey(ch))
	}
	m, cmd = m.Update(specialKey(tea.KeyTab))
	for _, ch := range image {
		m, cmd = m.Update(letterKey(ch))
	}
	m, cmd = m.Update(specialKey(tea.KeyEnter))

	return followRequest(t, cmd)
}

// A name that collides with an existing service is refused before
// cmds.AddService is ever dispatched - the write to disk that was always
// going to fail never happens.
func TestAddServiceModalRefusesADuplicateNameWithoutWriting(t *testing.T) {
	m := New("compose.yaml", []string{"web", "db"})

	msg := typeAndSubmit(t, m, "web", "nginx:alpine")

	if _, ok := msg.(cmds.AddServiceMsg); ok {
		t.Fatal("a duplicate name reached cmds.AddService")
	}
	errMsg, ok := msg.(cmds.OpenErrorModalMsg)
	if !ok {
		t.Fatalf("expected cmds.OpenErrorModalMsg, got %T", msg)
	}
	if !strings.Contains(errMsg.Message, "web") || !strings.Contains(errMsg.Message, "already exists") {
		t.Errorf("error message %q does not explain the collision", errMsg.Message)
	}
}

// A name that does not collide dispatches cmds.AddService with the typed
// name and image, and the fileName the modal was built with - run for real
// against a temp compose file, so this also confirms the fileName threads
// through the onSubmit closure correctly.
func TestAddServiceModalSubmitsANewName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "compose.yaml")
	if err := os.WriteFile(path, []byte("services:\n  web:\n    image: nginx:alpine\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := New(path, []string{"web"})

	msg := typeAndSubmit(t, m, "proxy", "traefik:v3")

	addMsg, ok := msg.(cmds.AddServiceMsg)
	if !ok {
		t.Fatalf("expected cmds.AddServiceMsg from AddService's cmd, got %T", msg)
	}
	if addMsg.Err != nil {
		t.Fatalf("AddService: %v", addMsg.Err)
	}
	if addMsg.ServiceName != "proxy" || addMsg.Image != "traefik:v3" {
		t.Errorf("got ServiceName=%q Image=%q, want proxy/traefik:v3", addMsg.ServiceName, addMsg.Image)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "proxy:") || !strings.Contains(string(contents), "traefik:v3") {
		t.Errorf("the new service was not written, got:\n%s", contents)
	}
}
