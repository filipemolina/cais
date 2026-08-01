package model

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components/composefilepickermodal"
	"github.com/filipemolina/stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

// onFilesPage is the app with a project loaded and the Files page active.
func onFilesPage(t *testing.T) AppModel {
	t.Helper()

	return drive(withGroupsLoaded(t), cmds.SetActivePageMsg("Compose Files"))
}

// b on the Files page asks for the directory of the active file to be
// scanned, which is what the picker lists.
func TestPressingBScansTheActiveFilesDirectory(t *testing.T) {
	m := onFilesPage(t)

	// b makes the panel ask for the picker; AppModel answers that intent by
	// issuing the directory scan. Two steps, as the runtime delivers them.
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'b', Text: "b"})
	m = updated.(AppModel)

	var scanCmd tea.Cmd
	for _, msg := range collect(cmd) {
		if _, ok := msg.(cmds.OpenComposeFilePickerMsg); ok {
			updated2, c := m.Update(msg)
			m = updated2.(AppModel)
			scanCmd = c
		}
	}
	if scanCmd == nil {
		t.Fatal("b did not reach the picker handler")
	}

	var scanned bool
	for _, c := range flattenCmds(scanCmd) {
		if msg := c(); msg != nil {
			if list, ok := msg.(cmds.ComposeFileListMsg); ok {
				scanned = true
				if list.Dir != "." {
					t.Errorf("scanned dir = %q, want %q (the active file's dir)", list.Dir, ".")
				}
			}
		}
	}
	if !scanned {
		t.Error("b did not scan the compose file's directory")
	}
}

// The scan result opens the picker modal.
func TestTheScanOpensThePicker(t *testing.T) {
	m := onFilesPage(t)

	updated, _ := m.Update(cmds.ComposeFileListMsg{
		Dir:   ".",
		Files: []string{"compose.yaml", "compose.yml"},
	})
	m = updated.(AppModel)

	if m.activeModal == nil {
		t.Fatal("the scan did not open a modal")
	}
	if _, ok := m.activeModal.(composefilepickermodal.Model); !ok {
		t.Fatalf("opened %T, want a composefilepickermodal.Model", m.activeModal)
	}
}

// A failed scan surfaces an error rather than opening an empty picker.
func TestAFailedScanShowsAnError(t *testing.T) {
	m := onFilesPage(t)

	updated, _ := m.Update(cmds.ComposeFileListMsg{Err: errBoom{}})
	m = updated.(AppModel)

	if m.activeModal != nil {
		t.Error("a failed scan opened a modal")
	}
	if m.lastError == "" {
		t.Error("a failed scan left no error")
	}
}

// The picker emits a switch request for the highlighted file, joining the
// scanned directory back onto the chosen name.
func TestThePickerSwitchesToTheHighlightedFile(t *testing.T) {
	m := onFilesPage(t)

	updated, _ := m.Update(cmds.ComposeFileListMsg{
		Dir:   ".",
		Files: []string{"compose.yaml", "compose.yml"},
	})
	m = updated.(AppModel)

	// Cursor starts on the active file (compose.yaml); move down to
	// compose.yml, then confirm.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = updated.(AppModel)

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(AppModel)

	// Enter closes the modal with the switch as its follow-up, so process
	// the close the way the runtime would.
	var followCmd tea.Cmd
	for _, msg := range collect(cmd) {
		if _, ok := msg.(cmds.CloseModalMsg); ok {
			updated2, closeCmd := m.Update(msg)
			m = updated2.(AppModel)
			followCmd = closeCmd
		}
	}

	var gotSwitch *cmds.SwitchComposeFileMsg
	for _, msg := range collect(followCmd) {
		if sw, ok := msg.(cmds.SwitchComposeFileMsg); ok {
			s := sw
			gotSwitch = &s
		}
	}
	if gotSwitch == nil {
		t.Fatal("Enter did not emit a SwitchComposeFileMsg")
	}
	if gotSwitch.Path != "compose.yml" {
		t.Errorf("switch path = %q, want %q", gotSwitch.Path, "compose.yml")
	}
}

// Switching points the source at the chosen file and reloads, exactly like
// passing --file at startup.
func TestSwitchingPointsTheSourceAtTheNewFile(t *testing.T) {
	m := onFilesPage(t)

	updated, cmd := m.Update(cmds.SwitchComposeFileMsg{Path: "compose.yml"})
	m = updated.(AppModel)

	if got, want := m.config.source, (utils.ComposeSource{File: "compose.yml"}); got != want {
		t.Errorf("source after switch = %+v, want %+v", got, want)
	}

	var reloaded bool
	for _, msg := range collect(cmd) {
		if _, ok := msg.(cmds.GetConfigMsg); ok {
			reloaded = true
		}
	}
	if !reloaded {
		t.Error("switching did not trigger a config reload")
	}
}

// Switching repaints the Files page with the new file's contents. The
// switch's reload (GetConfig) completes, which fires the contents read for
// the new file; the contents land and the viewport shows them - not the
// previous file's text. This is the full chain the runtime walks, driven
// end to end: the GetConfig and GetComposeFileContents commands are
// hand-fed their results (as withGroupsLoaded does) so the test needs no
// disk files, but every routing hop in between is real.
func TestSwitchingRepaintsTheFilesPageWithTheNewFile(t *testing.T) {
	m := onFilesPage(t)

	// Show the first file's contents, so there is something stale to replace.
	m = drive(m, cmds.ComposeFileContentsMsg{
		Name:     "compose.yaml",
		Contents: "services:\n  web:\n    image: nginx:alpine\n",
	})
	m = applyLayout(m)

	// The switch's reload completes for the new file. AppModel adopts the
	// new name and fires the contents read for it (recomposeFilesCmdIfActive
	// keys off the active page and configFileName, both now the new file).
	updated, cmd := m.Update(cmds.GetConfigMsg{
		FileName: "compose.yml",
		Files:    []string{"compose.yml"},
		Project:  groupProject(),
	})
	m = applyLayout(updated.(AppModel))

	var readsNew bool
	for _, c := range flattenCmds(cmd) {
		if msg := c(); msg != nil {
			if contents, ok := msg.(cmds.ComposeFileContentsMsg); ok {
				readsNew = true
				if contents.Name != "compose.yml" {
					t.Errorf("contents read for %q, want %q", contents.Name, "compose.yml")
				}
			}
		}
	}
	if !readsNew {
		t.Fatal("the switch's reload did not read the new file's contents")
	}

	// The contents arrive and the viewport shows them - redis, not the
	// nginx it was showing before the switch.
	m = drive(m, cmds.ComposeFileContentsMsg{
		Name:     "compose.yml",
		Contents: "services:\n  cache:\n    image: redis:alpine\n",
	})
	m = applyLayout(m)

	frame := ansi.Strip(m.View().Content)

	if !strings.Contains(frame, "redis:alpine") {
		t.Errorf("Files page does not show the new file's contents:\n%s", frame)
	}
	if strings.Contains(frame, "nginx:alpine") {
		t.Errorf("Files page still shows the previous file's contents after a switch:\n%s", frame)
	}
	if !strings.Contains(frame, "compose.yml") {
		t.Errorf("Files page does not name the new file:\n%s", frame)
	}
}
