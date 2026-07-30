package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components"
	"github.com/filipemolina/stack-stitcher/src/constants"
	"gopkg.in/yaml.v3"
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

// editingWeb puts the app in inline edit mode on the web service with the
// given fragment loaded, which is the starting state for the editor tests.
func editingWeb(t *testing.T, fragment string) AppModel {
	t.Helper()

	m := inlineEditProject(t)
	updated, cmd := m.Update(cmds.SetActivePageMsg("Services"))
	m = drive(updated, collect(cmd)...)

	// Focus the details panel and select web so the editor has a subject.
	details := constants.COMPONENT_BODY_DETAILS
	m = drive(m, collect(m.ChangeFocus(&details))...)
	m = drive(m, cmds.SetSelectedServiceMsg(types.ServiceConfig{Name: "web"}))

	// Enter edit mode.
	updated, cmd = m.Update(cmds.InlineEditReadyMsg{ServiceName: "web", Fragment: []byte(fragment)})
	m = drive(updated, collect(cmd)...)

	return m
}

// detailsPanel finds the Services page's DetailsPanelModel component.
func detailsPanel(t *testing.T, m AppModel) components.DetailsPanelModel {
	t.Helper()

	for _, component := range m.pages["Services"] {
		if panel, ok := component.(components.DetailsPanelModel); ok {
			return panel
		}
	}

	t.Fatal("no DetailsPanelModel found on the Services page")
	return components.DetailsPanelModel{}
}

// toEndOfPreviousLine moves the cursor up one row and to the end of it. The
// editor starts every test with the cursor at the end of the buffer, on the
// trailing empty row a fragment ending in "\n" leaves behind; this is the
// move to reach the last real line of text.
func toEndOfPreviousLine(m AppModel) AppModel {
	return drive(m, tea.KeyPressMsg{Code: tea.KeyUp}, tea.KeyPressMsg{Code: tea.KeyEnd})
}

func typeText(m AppModel, s string) AppModel {
	for _, r := range s {
		m = drive(m, letter(r))
	}
	return m
}

func TestEnterKeepsTheCurrentIndent(t *testing.T) {
	m := editingWeb(t, "web:\n  image: nginx\n")
	m = toEndOfPreviousLine(m)

	m = drive(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = typeText(m, "x")

	value := detailsPanel(t, m).EditorValue()
	if !strings.Contains(value, "  image: nginx\n  x") {
		t.Fatalf("expected the new line to keep the two-space indent, got: %q", value)
	}
}

func TestEnterDeepensAfterABlockOpener(t *testing.T) {
	m := editingWeb(t, "web:\n  ports:\n")
	m = toEndOfPreviousLine(m)

	m = drive(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = typeText(m, "-")

	value := detailsPanel(t, m).EditorValue()
	if !strings.Contains(value, "  ports:\n    -") {
		t.Fatalf("expected the dash to land at column 4, got: %q", value)
	}
}

func TestEnterAlignsInsideASequenceItem(t *testing.T) {
	m := editingWeb(t, "web:\n  environment:\n    - name: web\n")
	m = toEndOfPreviousLine(m)

	m = drive(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = typeText(m, "v")

	value := detailsPanel(t, m).EditorValue()
	if !strings.Contains(value, "    - name: web\n      v") {
		t.Fatalf("expected the new line to align under \"name\", got: %q", value)
	}
}

func TestEnterMidLineDoesNotDeepen(t *testing.T) {
	m := editingWeb(t, "web:\n  image: nginx\n")
	m = toEndOfPreviousLine(m)

	// "  image: nginx" - walk left off the end of "nginx" (5 characters) to
	// land the cursor right before the value.
	for range "nginx" {
		m = drive(m, tea.KeyPressMsg{Code: tea.KeyLeft})
	}

	m = drive(m, tea.KeyPressMsg{Code: tea.KeyEnter})

	value := detailsPanel(t, m).EditorValue()
	if !strings.Contains(value, "  image: \n  nginx") {
		t.Fatalf("expected a base-indent-only split with nothing lost, got: %q", value)
	}
	if !strings.Contains(value, "image:") || !strings.Contains(value, "nginx") {
		t.Fatalf("split lost part of the line, got: %q", value)
	}
}

func TestEnterKeepsTheDocumentParseable(t *testing.T) {
	m := editingWeb(t, "web:\n  image: nginx\n")
	m = toEndOfPreviousLine(m)

	m = drive(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = typeText(m, "ports:")
	m = drive(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m = typeText(m, `- "8080:80"`)

	panel := detailsPanel(t, m)
	value := panel.EditorValue()

	var doc any
	if err := yaml.Unmarshal([]byte(value), &doc); err != nil {
		t.Fatalf("built document is not valid YAML: %v\n%s", err, value)
	}

	view := ansi.Strip(m.View().Content)
	if !strings.Contains(view, "YAML ok") {
		t.Fatalf("expected the status line to say YAML ok, got: %s", view)
	}
}

func TestEnterOutsideEditModeIsUnchanged(t *testing.T) {
	m := inlineEditProject(t)
	updated, cmd := m.Update(cmds.SetActivePageMsg("Services"))
	m = drive(updated, collect(cmd)...)

	details := constants.COMPONENT_BODY_DETAILS
	m = drive(m, collect(m.ChangeFocus(&details))...)
	m = drive(m, cmds.SetSelectedServiceMsg(types.ServiceConfig{Name: "web"}))

	before := m

	updated, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = drive(updated.(AppModel), collect(cmd)...)

	if m.inlineEditing {
		t.Fatal("Enter should not enter edit mode on the details panel")
	}
	if detailsPanel(t, m).EditorValue() != detailsPanel(t, before).EditorValue() {
		t.Fatal("Enter outside edit mode should not touch the (closed) editor")
	}
}

func TestInlineEditingOwnsTheKeyboard(t *testing.T) {
	m := editingWeb(t, "web:\n  image: nginx\n")

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

func TestPasteLandsInTheEditor(t *testing.T) {
	m := editingWeb(t, "web:\n  image: nginx\n")

	updated, cmd := m.Update(tea.PasteMsg{Content: "  ports:\n    - \"8080:80\"\n"})
	m = drive(updated.(AppModel), collect(cmd)...)

	value := detailsPanel(t, m).EditorValue()
	if !strings.Contains(value, "8080:80") {
		t.Fatalf("pasted content missing from editor buffer: %q", value)
	}
}

func TestPasteKeepsItsOwnIndentation(t *testing.T) {
	m := editingWeb(t, "web:\n  image: nginx\n")

	updated, cmd := m.Update(tea.PasteMsg{Content: "  ports:\n    - \"8080:80\"\n"})
	m = drive(updated.(AppModel), collect(cmd)...)

	value := detailsPanel(t, m).EditorValue()
	if !strings.Contains(value, "  ports:") {
		t.Fatalf("pasted line lost its indentation: %q", value)
	}
	if !strings.Contains(value, "    - \"8080:80\"") {
		t.Fatalf("pasted continuation line lost its indentation: %q", value)
	}
}

func TestPasteRevalidates(t *testing.T) {
	m := editingWeb(t, "web:\n  image: nginx\n")

	updated, cmd := m.Update(tea.PasteMsg{Content: "\n  - [\n"})
	m = drive(updated.(AppModel), collect(cmd)...)

	view := ansi.Strip(m.View().Content)
	if !strings.Contains(view, "YAML:") {
		t.Fatalf("expected a YAML error in the status line after a bad paste, got: %s", view)
	}
}

func TestPasteOutsideEditModeIsInert(t *testing.T) {
	m := inlineEditProject(t)
	updated, cmd := m.Update(cmds.SetActivePageMsg("Services"))
	m = drive(updated, collect(cmd)...)

	details := constants.COMPONENT_BODY_DETAILS
	m = drive(m, collect(m.ChangeFocus(&details))...)
	m = drive(m, cmds.SetSelectedServiceMsg(types.ServiceConfig{Name: "web"}))

	updated, cmd = m.Update(tea.PasteMsg{Content: "  ports:\n    - \"8080:80\"\n"})
	m = drive(updated.(AppModel), collect(cmd)...)

	if m.inlineEditing {
		t.Fatal("paste should not enter edit mode on its own")
	}
}

// shiftTab is shift+tab as bubbletea v2 delivers it: the same code as tab,
// with the shift modifier.
func shiftTab() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
}

func lineAt(t *testing.T, value string, row int) string {
	t.Helper()

	lines := strings.Split(value, "\n")
	if row < 0 || row >= len(lines) {
		t.Fatalf("row %d out of range in %q", row, value)
	}
	return lines[row]
}

func TestTabIndentsTheCurrentLine(t *testing.T) {
	m := editingWeb(t, "web:\n  image: nginx\n")
	m = toEndOfPreviousLine(m)
	// Walk left off the end of "nginx" to land mid-line, before the value.
	for range "nginx" {
		m = drive(m, tea.KeyPressMsg{Code: tea.KeyLeft})
	}

	panel := detailsPanel(t, m)
	_, colBefore := panel.EditorCursor()
	before := lineAt(t, panel.EditorValue(), 1)

	m = drive(m, tea.KeyPressMsg{Code: tea.KeyTab})

	panel = detailsPanel(t, m)
	row, colAfter := panel.EditorCursor()
	after := lineAt(t, panel.EditorValue(), row)

	if after != "  "+before {
		t.Fatalf("expected two spaces added at the start of the line, got %q from %q", after, before)
	}
	if colAfter != colBefore+2 {
		t.Fatalf("expected the cursor to move right by 2, got %d -> %d", colBefore, colAfter)
	}
	if []rune(after)[colAfter] != []rune(before)[colBefore] {
		t.Fatalf("cursor no longer sits on the same character: before %q at %d, after %q at %d",
			before, colBefore, after, colAfter)
	}
}

func TestShiftTabOutdents(t *testing.T) {
	m := editingWeb(t, "web:\n    ports:\n")
	m = toEndOfPreviousLine(m)

	m = drive(m, shiftTab())

	got := lineAt(t, detailsPanel(t, m).EditorValue(), 1)
	if got != "  ports:" {
		t.Fatalf("expected the four-space line to outdent to two, got %q", got)
	}
}

func TestIndentThenOutdentIsARoundTrip(t *testing.T) {
	m := editingWeb(t, "web:\n  image: nginx\n")
	m = toEndOfPreviousLine(m)

	panel := detailsPanel(t, m)
	wantValue := panel.EditorValue()
	wantRow, wantCol := panel.EditorCursor()

	m = drive(m, tea.KeyPressMsg{Code: tea.KeyTab}, shiftTab())

	panel = detailsPanel(t, m)
	if got := panel.EditorValue(); got != wantValue {
		t.Fatalf("tab then shift+tab changed the buffer:\ngot  %q\nwant %q", got, wantValue)
	}
	if row, col := panel.EditorCursor(); row != wantRow || col != wantCol {
		t.Fatalf("tab then shift+tab left the cursor at (%d,%d), want (%d,%d)", row, col, wantRow, wantCol)
	}
}

func TestOutdentAtColumnZeroIsANoOp(t *testing.T) {
	m := editingWeb(t, "web:\n")
	m = drive(m, tea.KeyPressMsg{Code: tea.KeyUp}, tea.KeyPressMsg{Code: tea.KeyHome})

	before := detailsPanel(t, m).EditorValue()

	m = drive(m, shiftTab())

	after := detailsPanel(t, m).EditorValue()
	if after != before {
		t.Fatalf("outdenting an unindented line should be a no-op, got %q from %q", after, before)
	}
}

func TestOutdentOfAPartialIndentClampsToZero(t *testing.T) {
	m := editingWeb(t, "web:\n   image: nginx\n")
	m = toEndOfPreviousLine(m)

	m = drive(m, shiftTab())

	got := lineAt(t, detailsPanel(t, m).EditorValue(), 1)
	if got != " image: nginx" {
		t.Fatalf("expected the three-space line to outdent to one space, got %q", got)
	}
}

func TestBackspaceInIndentEatsALevel(t *testing.T) {
	m := editingWeb(t, "web:\n    ports:\n")
	m = drive(m, tea.KeyPressMsg{Code: tea.KeyUp}, tea.KeyPressMsg{Code: tea.KeyHome})
	for range 4 {
		m = drive(m, tea.KeyPressMsg{Code: tea.KeyRight})
	}

	m = drive(m, tea.KeyPressMsg{Code: tea.KeyBackspace})

	panel := detailsPanel(t, m)
	got := lineAt(t, panel.EditorValue(), 1)
	if got != "  ports:" {
		t.Fatalf("expected backspace to eat one indent level, got %q", got)
	}
	if row, col := panel.EditorCursor(); row != 1 || col != 2 {
		t.Fatalf("expected the cursor at (1,2), got (%d,%d)", row, col)
	}
}

func TestBackspaceInTextIsUnchanged(t *testing.T) {
	m := editingWeb(t, "web:\n  image: nginx\n")
	m = toEndOfPreviousLine(m)

	m = drive(m, tea.KeyPressMsg{Code: tea.KeyBackspace})

	got := lineAt(t, detailsPanel(t, m).EditorValue(), 1)
	if got != "  image: ngin" {
		t.Fatalf("expected backspace in text to delete exactly one character, got %q", got)
	}
}

func TestBackspaceAtColumnZeroStillMergesLines(t *testing.T) {
	m := editingWeb(t, "web:\n  image: nginx\n")
	m = drive(m, tea.KeyPressMsg{Code: tea.KeyUp}, tea.KeyPressMsg{Code: tea.KeyHome})

	m = drive(m, tea.KeyPressMsg{Code: tea.KeyBackspace})

	value := detailsPanel(t, m).EditorValue()
	if !strings.HasPrefix(value, "web:  image: nginx") {
		t.Fatalf("expected backspace at column 0 to merge with the line above, got %q", value)
	}
}

func TestIndentKeysDoNotSwitchPanelsWhileEditing(t *testing.T) {
	m := editingWeb(t, "web:\n  image: nginx\n")
	focusedBefore := m.focusedComponent

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = drive(updated.(AppModel), collect(cmd)...)

	if m.focusedComponent != focusedBefore {
		t.Fatalf("tab moved focus from %d to %d while editing", focusedBefore, m.focusedComponent)
	}
}

func TestTabStillSwitchesPanelsOutsideTheEditor(t *testing.T) {
	m := inlineEditProject(t)
	updated, cmd := m.Update(cmds.SetActivePageMsg("Services"))
	m = drive(updated, collect(cmd)...)
	focusedBefore := m.focusedComponent

	updated, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = drive(updated.(AppModel), collect(cmd)...)

	if m.focusedComponent == focusedBefore {
		t.Fatal("tab should still move focus when the editor is not open")
	}
}
