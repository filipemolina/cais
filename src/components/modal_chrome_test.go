package components

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/components/aboutmodal"
	"github.com/filipemolina/stack-stitcher/src/components/confirmmodal"
	"github.com/filipemolina/stack-stitcher/src/components/errormodal"
	"github.com/filipemolina/stack-stitcher/src/components/groupslist"
	"github.com/filipemolina/stack-stitcher/src/components/helpoverlay"
	"github.com/filipemolina/stack-stitcher/src/components/serviceslist"
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
		{"about", aboutmodal.New(), "stack-stitcher", "esc"},
		{"help", helpoverlay.New(keys.Context{Page: "Home"}, nil, 100), "Keyboard shortcuts", "esc"},
		{"confirm", confirmmodal.New("Delete group \"core\"?", nil), "Confirm", "esc"},
		{"error", errormodal.New("boom", 100), "Error", "esc"},
		{"group name", GroupNameModal(nil, []string{"web"}, 40), "New group", "esc"},
		{"rename group", GroupNameModalForRename("core", nil), "Rename group", "esc"},
		{"service checklist", ServiceChecklistModal("core", []string{"web"}, 40), "Select services", "esc"},
		{"edit group members", ServiceChecklistModalForEdit("core", []string{"web"}, []string{"web"}, 40), "Edit members", "esc"},
		{"create compose file", CreateComposeFileModal("."), "New compose file", "esc"},
		{"compose file picker", ComposeFilePickerModal(".", []string{"compose.yaml"}, "compose.yaml", 40), "Switch compose file", "esc"},
		{"theme picker", ThemePickerModal(40), "Choose theme", "esc"},
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

// A modal whose body is a list is the one shape that can outgrow the screen:
// the item count comes from the user's project, not from the code. View.go's
// renderWithModal clamps a modal's y to 0 rather than scrolling it, so an
// oversized modal loses its hint line and bottom border off the bottom edge
// with no way to reach them. Every list-backed modal has to size itself to
// the terminal instead of to len(items).
func TestListModalsFitAShortTerminal(t *testing.T) {
	const termHeight = 24

	// More items than a 24-row terminal can hold, whichever modal they land in.
	many := make([]string, 40)
	for i := range many {
		many[i] = fmt.Sprintf("service-%02d", i)
	}

	cases := []struct {
		name  string
		modal tea.Model
	}{
		{"compose file picker", ComposeFilePickerModal(".", many, many[0], termHeight)},
		{"service checklist", ServiceChecklistModal("core", many, termHeight)},
		{"edit group members", ServiceChecklistModalForEdit("core", many, many[:2], termHeight)},
		{"theme picker", ThemePickerModal(termHeight)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if h := lipgloss.Height(tc.modal.View().Content); h > termHeight {
				t.Errorf("modal is %d rows tall in a %d-row terminal", h, termHeight)
			}
		})
	}
}

// The panel lists are constructed once in AppModel and never rebuilt, so any
// color baked into the list's own Styles at construction survives a theme
// switch and leaves the title chip painted in the startup theme while the
// rest of the frame repaints. This is the guard for that: the chip's fill has
// to be the *currently* active accent, whichever theme was active when the
// list was built.
func TestListTitleChipFollowsTheActiveTheme(t *testing.T) {
	defer appstyles.SetTheme(appstyles.DefaultTheme)

	// The truecolor background SGR lipgloss emits for the active accent.
	accentFill := func() string {
		r, g, b, _ := appstyles.Active.Accent.RGBA()
		return fmt.Sprintf("48;2;%d;%d;%d", r>>8, g>>8, b>>8)
	}

	// Built under the default theme, as AppModel builds them at startup.
	lists := map[string]struct {
		model tea.Model
		title string
	}{
		"groups":   {groupslist.New([]string{"core", "edge"}, 60, 20), "Groups"},
		"services": {serviceslist.New([]types.ServiceConfig{{Name: "web"}}, 60, 20), "Services"},
	}

	for name, tc := range lists {
		t.Run(name, func(t *testing.T) {
			// Switch away from the theme the list was constructed under.
			appstyles.SetTheme("gruvbox-dark")

			var titleRow string
			for _, line := range strings.Split(tc.model.View().Content, "\n") {
				if strings.Contains(ansi.Strip(line), tc.title) {
					titleRow = line
					break
				}
			}
			if titleRow == "" {
				t.Fatalf("no %q title row in the rendered list", tc.title)
			}

			if want := accentFill(); !strings.Contains(titleRow, want) {
				t.Errorf("title chip is not filled with the active accent (%s) - a style is frozen:\n%q",
					want, titleRow)
			}
		})
	}
}
