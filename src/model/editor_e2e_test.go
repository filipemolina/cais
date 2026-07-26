package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/constants"

	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

const editorFixture = `services:
  web:
    image: nginx:alpine
    profiles: ["core"] # front door
    ports:
      - "8085:80"

  db:
    image: postgres:alpine
    profiles: ["core"]
`

// scriptEditor writes a shell script standing in for the user's editor and
// points the environment at it. body receives the file to edit as "$1".
func scriptEditor(t *testing.T, dir, body string) {
	t.Helper()

	path := filepath.Join(dir, "editor.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatalf("writing the stand-in editor: %v", err)
	}

	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", path)
}

// composeProject drops a compose file in a temp directory and makes it the
// working directory, which is where the app looks for one.
func composeProject(t *testing.T, contents string) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(contents), 0o644); err != nil {
		t.Fatalf("writing the compose fixture: %v", err)
	}
	t.Chdir(dir)

	return dir
}

// The keypress half of the feature: E on a focused details panel asks for
// the editor. Driven through the model rather than the rig, because the two
// halves meet at this message and this half needs no timing at all.
func TestPressingEAsksForTheEditor(t *testing.T) {
	m := drive(applyLayout(startup(120, 40)), cmds.SetActivePageMsg("Dashboard"))
	m = drive(m,
		cmds.SetServicesListMsg([]types.ServiceConfig{{Name: "web"}}),
		cmds.SetSelectedServiceMsg(types.ServiceConfig{Name: "web"}),
	)

	details := constants.COMPONENT_BODY_DETAILS
	m = drive(m, collect(m.ChangeFocus(&details))...)

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'E', Text: "E", Mod: tea.ModShift})

	for _, msg := range collect(cmd) {
		if _, ok := msg.(cmds.OpenEditorMsg); ok {
			return
		}
	}

	t.Error("E on the details panel did not ask for the editor")
}

// The other half, end to end against a real editor process: the app hands
// over the terminal, waits for the editor to exit, and reloads whatever was
// saved. tea.ExecProcess does the handover for real here - the rig has no
// TTY, but the process still runs and the program still suspends and
// resumes around it.
func TestEditingTheComposeFileInAnExternalEditor(t *testing.T) {
	dir := composeProject(t, editorFixture)
	scriptEditor(t, dir, `sed -i 's|nginx:alpine|nginx:1.28|' "$1"`)

	r := newRig(t)
	if !r.WaitFor("web", 3*time.Second) {
		t.Fatal("the services never rendered")
	}

	r.Send(cmds.OpenEditorMsg{})

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		saved, err := os.ReadFile(filepath.Join(dir, "compose.yaml"))
		if err == nil && strings.Contains(string(saved), "nginx:1.28") {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatal("the editor never ran against the compose file")
}

// An editor that cannot be started must not leave the app believing it is
// suspended - background refreshes would stay off for the rest of the run.
func TestAnEditorThatCannotStartDoesNotWedgeTheApp(t *testing.T) {
	dir := composeProject(t, editorFixture)
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", filepath.Join(dir, "does-not-exist"))

	r := newRig(t)
	if !r.WaitFor("web", 3*time.Second) {
		t.Fatal("the services never rendered")
	}

	r.Send(cmds.OpenEditorMsg{})

	// The failure has to come back as EditorClosedMsg carrying the error,
	// which is what clears the suspended flag. Nothing observable happens on
	// screen, so this asserts the app is still alive and rendering.
	time.Sleep(500 * time.Millisecond)
	r.Send(tea.WindowSizeMsg{Width: 100, Height: 30})

	if !r.WaitFor("web", 3*time.Second) {
		t.Fatal("the app stopped rendering after a failed editor launch")
	}
}

// Phase 2 end to end: e opens just the selected service, and what comes
// back is spliced into the compose file.
func TestEditingASingleServiceInAnExternalEditor(t *testing.T) {
	dir := composeProject(t, editorFixture)
	// The scratch file holds only "web:", so this rewrite cannot touch db.
	scriptEditor(t, dir, `sed -i 's|nginx:alpine|nginx:1.28|' "$1"`)

	r := newRig(t)
	if !r.WaitFor("web", 3*time.Second) {
		t.Fatal("the services never rendered")
	}

	r.Send(cmds.OpenServiceEditorMsg{ServiceName: "web"})

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		saved := readComposeFile(t, dir)
		if strings.Contains(saved, "nginx:1.28") {
			if !strings.Contains(saved, "# front door") {
				t.Errorf("the edit lost the service's comment:\n%s", saved)
			}
			if !strings.Contains(saved, "postgres:alpine") {
				t.Errorf("the edit disturbed a neighbouring service:\n%s", saved)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("the service was never edited:\n%s", readComposeFile(t, dir))
}

// A rejected edit has to leave the file alone and leave the app usable.
func TestAnInvalidServiceEditIsRefused(t *testing.T) {
	dir := composeProject(t, editorFixture)
	before := readComposeFile(t, dir)
	scriptEditor(t, dir, `printf 'web: not-a-mapping\n' > "$1"`)

	r := newRig(t)
	if !r.WaitFor("web", 3*time.Second) {
		t.Fatal("the services never rendered")
	}

	r.Send(cmds.OpenServiceEditorMsg{ServiceName: "web"})
	time.Sleep(1 * time.Second)

	if after := readComposeFile(t, dir); after != before {
		t.Errorf("a refused edit changed the file:\n%s", after)
	}

	// Still alive and rendering, with no editor loop to escape.
	r.Send(tea.WindowSizeMsg{Width: 100, Height: 30})
	if !r.WaitFor("web", 3*time.Second) {
		t.Fatal("the app stopped rendering after a refused edit")
	}
}

// Quitting the editor without saving cancels, writing nothing at all.
func TestAnUnchangedServiceEditWritesNothing(t *testing.T) {
	dir := composeProject(t, editorFixture)
	before := readComposeFile(t, dir)
	scriptEditor(t, dir, `true`)

	r := newRig(t)
	if !r.WaitFor("web", 3*time.Second) {
		t.Fatal("the services never rendered")
	}

	r.Send(cmds.OpenServiceEditorMsg{ServiceName: "web"})
	time.Sleep(1 * time.Second)

	if after := readComposeFile(t, dir); after != before {
		t.Errorf("an unchanged edit rewrote the file:\n%s", after)
	}

	// The scratch file is removed however the edit ends.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".web.") {
			t.Errorf("the scratch file was left behind: %s", entry.Name())
		}
	}
}

func readComposeFile(t *testing.T, dir string) string {
	t.Helper()

	contents, err := os.ReadFile(filepath.Join(dir, "compose.yaml"))
	if err != nil {
		t.Fatalf("reading the compose file: %v", err)
	}

	return string(contents)
}
