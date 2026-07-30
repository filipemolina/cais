package model

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/filipemolina/stack-stitcher/src/cmds"
)

// openHelp presses ? and drives the resulting message back into the model,
// the way the runtime would deliver it.
func openHelp(t *testing.T, m AppModel) AppModel {
	t.Helper()

	updated, cmd := m.Update(letter('?'))

	var opened bool
	for _, msg := range collect(cmd) {
		if _, ok := msg.(cmds.OpenHelpModalMsg); ok {
			opened = true
		}
	}
	if !opened {
		t.Fatal("? produced no OpenHelpModalMsg")
	}

	m = drive(updated, collect(cmd)...)
	if m.activeModal == nil {
		t.Fatal("? did not open the help overlay")
	}

	return m
}

// ? opens the overlay, and each of ?, esc and q closes it again. q closes
// only the overlay - it must never quit the app from under it.
func TestHelpOverlayOpensAndCloses(t *testing.T) {
	closers := map[string]tea.KeyPressMsg{
		"?":   letter('?'),
		"esc": escKey(),
		"q":   letter('q'),
	}

	for name, stroke := range closers {
		t.Run(name, func(t *testing.T) {
			m := openHelp(t, homeWithGroups(t))

			updated, cmd := m.Update(stroke)

			var closed bool
			for _, msg := range collect(cmd) {
				switch msg.(type) {
				case cmds.CloseModalMsg:
					closed = true
				case tea.QuitMsg:
					t.Fatalf("%s quit the app from inside the help overlay", name)
				}
			}
			if !closed {
				t.Fatalf("%s did not close the help overlay", name)
			}

			m = drive(updated, collect(cmd)...)
			if m.activeModal != nil {
				t.Fatalf("%s left a modal open", name)
			}
		})
	}
}

// While a filter is being typed, ? is a question mark: the filter input gets
// it and no overlay opens.
func TestQuestionMarkIsALetterWhileFiltering(t *testing.T) {
	m := filtering(t, homeWithGroups(t))

	_, cmd := m.Update(letter('?'))

	for _, msg := range collect(cmd) {
		if _, ok := msg.(cmds.OpenHelpModalMsg); ok {
			t.Fatal("? opened the help overlay while a filter was being typed")
		}
	}
}

// The overlay renders what the keymap says - the scopes and the rows - and,
// when several compose files were in the running, the candidates that lost.
func TestHelpOverlayRendersTheCatalog(t *testing.T) {
	m := applyLayout(drive(startup(120, 40),
		cmds.GetConfigMsg{
			FileName: "compose.yaml",
			Files:    []string{"compose.yaml", "compose.yml", "docker-compose.yml"},
			Project:  project(),
		},
	))
	m = openHelp(t, m)

	overlay := ansi.Strip(m.activeModal.View().Content)

	for _, want := range []string{
		"Keyboard shortcuts",
		"Pages", "List", "Details", "Editor", "Overlays", "Global",
		"space start",
		"alt+g/s/f",
		"Compose files",
		"compose.yml",
		"docker-compose.yml",
	} {
		if !strings.Contains(overlay, want) {
			t.Errorf("help overlay does not mention %q", want)
		}
	}
}
