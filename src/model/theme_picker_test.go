package model

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// TestRigThemePickerOpens presses T (shift+t) and checks the picker
// renders, then Esc closes it. Theme restoration under Esc is tested
// without concurrent access in TestThemePickerEscRestoresOriginalTheme
// (the component-level test); touching appstyles.Active from the test
// goroutine while the program goroutine is rendering would be a data race.
func TestRigThemePickerOpens(t *testing.T) {
	setupProjectDir(t)

	r := newRig(t)
	if !r.WaitFor("core", 3*time.Second) {
		t.Fatal("app never rendered the project")
	}

	// T (shift+t) opens the theme picker. The binding matches on "T",
	// which is the Text field of a KeyPressMsg.
	r.Send(tea.KeyPressMsg{Code: 'T', Text: "T"})

	if !r.WaitFor("Choose theme", 3*time.Second) {
		t.Fatalf("theme picker did not open. Output:\n%s", r.Output())
	}

	// Esc closes the picker.
	r.Send(tea.KeyPressMsg{Code: tea.KeyEsc})

	if !r.WaitForNot("Choose theme", 3*time.Second) {
		t.Fatalf("theme picker did not close after Esc. Output:\n%s", r.Output())
	}
}
