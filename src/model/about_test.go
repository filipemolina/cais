package model

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/filipemolina/stack-stitcher/src/components/aboutmodal"

	tea "charm.land/bubbletea/v2"
)

// a opens the About modal, the same message path every other modal takes.
func TestPressingAOpensTheAboutModal(t *testing.T) {
	m := withGroupsLoaded(t)

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	m = drive(updated.(AppModel), collect(cmd)...)

	if m.activeModal == nil {
		t.Fatal("a did not open a modal")
	}
	if _, ok := m.activeModal.(aboutmodal.Model); !ok {
		t.Fatalf("a opened %T, want an aboutmodal.Model", m.activeModal)
	}
}

// The About modal closes on esc, as every overlay does. While a modal is open
// it owns the keyboard, so esc reaches it via AppModel.Update's modal path and
// comes back as a CloseModal; driving that the way the runtime does clears
// activeModal.
func TestEscClosesTheAboutModal(t *testing.T) {
	m := withGroupsLoaded(t)

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	m = drive(updated.(AppModel), collect(cmd)...)
	if m.activeModal == nil {
		t.Fatal("precondition: a did not open the About modal")
	}

	updated, cmd = m.Update(keyPress(teaKeyEsc()))
	m = drive(updated.(AppModel), collect(cmd)...)

	if m.activeModal != nil {
		t.Error("esc did not close the About modal")
	}
}

// The About modal shows the wordmark, the version, the license and the repo
// link - the four things the TODO asked it to carry.
func TestTheAboutModalShowsVersionLicenseAndRepo(t *testing.T) {
	m := withGroupsLoaded(t)

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	m = drive(updated.(AppModel), collect(cmd)...)
	if m.activeModal == nil {
		t.Fatal("precondition: a did not open the About modal")
	}

	frame := ansi.Strip(m.activeModal.View().Content)

	for _, want := range []string{"Stack Stitcher", "version", "MIT", "github.com/filipemolina/stack-stitcher"} {
		if !strings.Contains(frame, want) {
			t.Errorf("About modal does not show %q:\n%s", want, frame)
		}
	}
}
