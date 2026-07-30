package components

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

// Every modal takes over the keyboard and hides the footer bar behind it, so
// each one has to say what it is (a title) and how to get out (a hint line).
// This is the regression guard for both: a new modal that skips either shows
// up here rather than as a user stuck on an unlabelled box.
func TestEveryModalHasATitleAndAnExitHint(t *testing.T) {
	logs, _ := LogsModal("web", false, "compose.yaml", 100, 40)

	cases := []struct {
		name  string
		modal tea.Model
		// title is a distinctive substring of the modal's heading.
		title string
		// exitKey is the key the modal's hint line must advertise as the way
		// out. Every modal answers esc; About and Help also close on q.
		exitKey string
	}{
		{"about", AboutModal(), "stack-stitcher", "esc"},
		{"help", HelpOverlay(keys.Context{Page: "Home"}, nil, 100), "Keyboard shortcuts", "esc"},
		{"confirm", ConfirmModal("Delete group \"core\"?", nil), "Confirm", "esc"},
		{"error", ErrorModal("boom", 100), "Error", "esc"},
		{"group name", GroupNameModal(nil, []string{"web"}), "New group", "esc"},
		{"service checklist", ServiceChecklistModal("core", []string{"web"}), "Select services", "esc"},
		{"edit group members", ServiceChecklistModalForEdit("core", []string{"web"}, []string{"web"}), "Edit members", "esc"},
		{"create compose file", CreateComposeFileModal("."), "New compose file", "esc"},
		{"compose file picker", ComposeFilePickerModal(".", []string{"compose.yaml"}, "compose.yaml"), "Switch compose file", "esc"},
		{"theme picker", ThemePickerModal(), "Choose theme", "esc"},
		{"logs", logs, "logs: web", "esc"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			frame := ansi.Strip(tc.modal.View().Content)

			if !strings.Contains(frame, tc.title) {
				t.Errorf("modal is missing its title %q:\n%s", tc.title, frame)
			}
			if !strings.Contains(frame, tc.exitKey) {
				t.Errorf("modal never advertises %q as the way out:\n%s", tc.exitKey, frame)
			}
		})
	}
}

// TestCreateComposeFileModalHintsEveryStep covers the one modal with more
// than one screen: each step advertises the keys that step actually answers.
func TestCreateComposeFileModalHintsEveryStep(t *testing.T) {
	m := CreateComposeFileModal(".")

	// Step 1: filename. Enter advances rather than creating anything.
	frame := ansi.Strip(m.View().Content)
	if !strings.Contains(frame, "next") {
		t.Errorf("filename step does not say enter advances:\n%s", frame)
	}

	// Enter with the default filename moves to the add-a-service prompt.
	m, _ = m.Update(specialKey(tea.KeyEnter))
	frame = ansi.Strip(m.View().Content)
	for _, want := range []string{"y", "n", "esc"} {
		if !strings.Contains(frame, want) {
			t.Errorf("add-service prompt does not advertise %q:\n%s", want, frame)
		}
	}

	// y opens the service fields, which are two inputs plus a create.
	m, _ = m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	frame = ansi.Strip(m.View().Content)
	for _, want := range []string{"next field", "create file", "esc"} {
		if !strings.Contains(frame, want) {
			t.Errorf("service fields step does not advertise %q:\n%s", want, frame)
		}
	}
}
