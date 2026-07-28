package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/constants"
)

const inlineEditFixture = `services:
  web:
    image: nginx:alpine
    profiles: ["core"]
  api:
    image: node:alpine
`

func inlineEditProject(t *testing.T) AppModel {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(inlineEditFixture), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	t.Chdir(dir)

	m := startup(120, 40)
	updated, cmd := m.Update(cmds.GetConfigMsg{FileName: "compose.yaml", Project: &types.Project{
		Services: types.Services{
			"web": types.ServiceConfig{Name: "web", Profiles: []string{"core"}},
			"api": types.ServiceConfig{Name: "api"},
		},
	}})
	m = drive(updated, collect(cmd)...)
	m = applyLayout(m)

	return m
}

func TestRequestInlineEditReturnsFragment(t *testing.T) {
	m := inlineEditProject(t)

	updated, cmd := m.Update(cmds.SetActivePageMsg("Services"))
	m = drive(updated, collect(cmd)...)

	updated, cmd = m.Update(cmds.RequestInlineEditMsg{ServiceName: "web"})
	m = updated.(AppModel)

	var ready cmds.InlineEditReadyMsg
	for _, msg := range collect(cmd) {
		if r, ok := msg.(cmds.InlineEditReadyMsg); ok {
			ready = r
		}
	}

	if ready.ServiceName != "web" {
		t.Fatalf("expected inline edit ready for web, got %q", ready.ServiceName)
	}
	if ready.Err != nil {
		t.Fatalf("expected no fragment error, got %v", ready.Err)
	}
	if !strings.Contains(string(ready.Fragment), "image: nginx:alpine") {
		t.Fatalf("fragment missing the original image: %s", ready.Fragment)
	}
}

func TestRequestInlineEditWithoutFileReportsError(t *testing.T) {
	m := startup(120, 40)

	updated, cmd := m.Update(cmds.RequestInlineEditMsg{ServiceName: "web"})
	m = updated.(AppModel)

	var ready cmds.InlineEditReadyMsg
	for _, msg := range collect(cmd) {
		if r, ok := msg.(cmds.InlineEditReadyMsg); ok {
			ready = r
		}
	}

	if ready.Err == nil {
		t.Fatal("expected a fragment error when no file is loaded")
	}
	if m.lastError != "" {
		t.Fatal("inline edit fragment error should not reach the banner")
	}
}

func TestRequestSaveServiceWritesAndReloads(t *testing.T) {
	m := inlineEditProject(t)

	fragment := []byte("web:\n  image: nginx:1.28\n  profiles: [\"core\"]\n")
	updated, cmd := m.Update(cmds.RequestSaveServiceMsg{ServiceName: "web", Fragment: fragment})
	m = updated.(AppModel)

	// Running the command performs the write and produces ServiceSavedMsg.
	// Re-process that message so the success handler queues the reload.
	var saved cmds.ServiceSavedMsg
	var reloaded bool
	for _, msg := range collect(cmd) {
		if s, ok := msg.(cmds.ServiceSavedMsg); ok {
			saved = s
			updated, saveCmd := m.Update(s)
			m = updated.(AppModel)
			for _, follow := range collect(saveCmd) {
				if _, ok := follow.(cmds.GetConfigMsg); ok {
					reloaded = true
				}
			}
		}
	}

	if saved.ServiceName != "web" {
		t.Fatalf("expected save result for web, got %q", saved.ServiceName)
	}
	if saved.Err != nil {
		t.Fatalf("save failed: %v", saved.Err)
	}
	if m.lastError != "" {
		t.Fatalf("successful inline save should not reach the banner: %q", m.lastError)
	}
	if !reloaded {
		t.Fatal("successful inline save should queue a config reload")
	}

	// The file on disk should reflect the change.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting cwd: %v", err)
	}
	savedContents, err := os.ReadFile(filepath.Join(cwd, "compose.yaml"))
	if err != nil {
		t.Fatalf("reading the saved file: %v", err)
	}
	if !strings.Contains(string(savedContents), "nginx:1.28") {
		t.Fatalf("the compose file was not updated: %s", savedContents)
	}
	if strings.Contains(string(savedContents), "nginx:alpine") {
		t.Fatalf("the old image survived in the file: %s", savedContents)
	}
}

func TestRequestSaveServiceFailureDoesNotReload(t *testing.T) {
	m := inlineEditProject(t)

	// Missing the service name header; the loader should reject this.
	fragment := []byte("image: nginx:1.28\n")
	updated, cmd := m.Update(cmds.RequestSaveServiceMsg{ServiceName: "web", Fragment: fragment})
	m = updated.(AppModel)

	m = drive(m, collect(cmd)...)

	var saved cmds.ServiceSavedMsg
	var reloaded bool
	for _, msg := range collect(cmd) {
		if s, ok := msg.(cmds.ServiceSavedMsg); ok {
			saved = s
		}
		if _, ok := msg.(cmds.GetConfigMsg); ok {
			reloaded = true
		}
	}

	if saved.Err == nil {
		t.Fatal("expected save to fail with a malformed fragment")
	}
	if m.lastError != "" {
		t.Fatalf("inline save error should stay inline; banner: %q", m.lastError)
	}
	if reloaded {
		t.Fatal("failed inline save should not queue a reload")
	}
}

func TestInlineEditingOwnsTheKeyboard(t *testing.T) {
	m := inlineEditProject(t)
	updated, cmd := m.Update(cmds.SetActivePageMsg("Services"))
	m = drive(updated, collect(cmd)...)

	// Focus the details panel and select web so the editor has a subject.
	details := constants.COMPONENT_BODY_DETAILS
	m = drive(m, collect(m.ChangeFocus(&details))...)
	m = drive(m, cmds.SetSelectedServiceMsg(types.ServiceConfig{Name: "web"}))

	// Enter edit mode.
	updated, cmd = m.Update(cmds.InlineEditReadyMsg{ServiceName: "web", Fragment: []byte("web:\n  image: nginx\n")})
	m = drive(updated, collect(cmd)...)

	if !m.inlineEditing {
		t.Fatal("AppModel should record inline editing")
	}
	if !m.keyboardOwned() {
		t.Fatal("keyboard should be owned while editing")
	}

	// A page digit should be ignored while editing.
	updated, pageCmd := m.Update(tea.KeyPressMsg{Code: '1', Text: "1"})
	m = updated.(AppModel)
	if activePageFrom(collect(pageCmd)) != "" {
		t.Fatal("page digit should be inert while editing")
	}

	// The help context should advertise editor keys.
	ctx := m.helpContext()
	if !ctx.Editing {
		t.Fatal("help context should report editing")
	}
}
