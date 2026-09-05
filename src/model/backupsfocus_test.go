package model

import (
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/keys"
	"github.com/filipemolina/cais/src/utils"
)

// onBackups is the app sitting on the Backups page with several real copies
// in the store and both panels sized, settled the way the runtime settles it.
//
// Each copy gets distinct content on purpose: the store deduplicates by
// content hash, so snapshotting the same bytes twice leaves one row and a
// cursor with nowhere to move.
func onBackups(t *testing.T) AppModel {
	t.Helper()

	m := withComposeLoaded(t)

	for _, image := range []string{"nginx:1.25", "nginx:1.26", "nginx:1.27"} {
		writeFile(t, m.config.configFileName, "services:\n  app:\n    image: "+image+"\n")
		if err := utils.SnapshotFile(m.config.configFileName); err != nil {
			t.Fatalf("seeding a backup: %v", err)
		}
	}

	return settle(t, m,
		tea.WindowSizeMsg{Width: 120, Height: 40},
		cmds.SetActivePageMsg("Backups"),
	)
}

var tabKey = tea.KeyPressMsg{Code: tea.KeyTab}
var shiftTabKey = tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}

// Tab moves focus between the two Backups panels, and AppModel is the one
// handler for it. Both panels receive every keystroke, so a page where each
// panel decided its own focus is a page that ends up with both halves lit or
// neither.
func TestTabMovesFocusBetweenTheBackupsPanels(t *testing.T) {
	m := onBackups(t)

	if m.backupsFocus != apptypes.BackupsList {
		t.Fatalf("the page should open on the list, got %v", m.backupsFocus)
	}

	m = settle(t, m, tabKey)
	if m.backupsFocus != apptypes.BackupsPreview {
		t.Errorf("tab left focus on %v, want the preview", m.backupsFocus)
	}

	m = settle(t, m, tabKey)
	if m.backupsFocus != apptypes.BackupsList {
		t.Errorf("a second tab left focus on %v, want the list again", m.backupsFocus)
	}
}

// With exactly two stops, shift+tab is the same move as tab. It is bound
// anyway because a user who reaches for it expects it to work, not to be a
// dead key that happens to have nowhere else to go.
func TestShiftTabAlsoMovesBackupsFocus(t *testing.T) {
	m := settle(t, onBackups(t), shiftTabKey)

	if m.backupsFocus != apptypes.BackupsPreview {
		t.Errorf("shift+tab left focus on %v, want the preview", m.backupsFocus)
	}
}

// Tab is dead everywhere else. Home and Services gave up focus because their
// two panels are two views of one selection; reviving tab there would resurrect
// the abstraction docs/DESIGN.md removed.
func TestTabIsStillDeadOnTheOtherPages(t *testing.T) {
	for _, page := range []string{"Home", "Services", "Compose Files"} {
		m := settle(t, withComposeLoaded(t),
			tea.WindowSizeMsg{Width: 120, Height: 40},
			cmds.SetActivePageMsg(page),
		)

		tabbed := settle(t, m, tabKey)
		if tabbed.backupsFocus != apptypes.BackupsList {
			t.Errorf("tab on %s moved the Backups focus to %v", page, tabbed.backupsFocus)
		}
	}
}

// Leaving the page with the preview lit and coming back has to land on the
// list again. Without the reset the panels keep their stale flags and the page
// reopens with the arrows driving a preview the cursor never picked a row for.
func TestLeavingAndReturningResetsBackupsFocus(t *testing.T) {
	m := settle(t, onBackups(t), tabKey)
	if m.backupsFocus != apptypes.BackupsPreview {
		t.Fatal("precondition: the preview should hold focus before leaving the page")
	}

	m = settle(t, m, cmds.SetActivePageMsg("Home"))
	m = settle(t, m, cmds.SetActivePageMsg("Backups"))

	if m.backupsFocus != apptypes.BackupsList {
		t.Errorf("returning to Backups left focus on %v, want the list", m.backupsFocus)
	}

	// The reset has to reach the panels, not just AppModel's own field: they
	// route their keys by their own copy of it, and a stale copy is a page
	// where the arrows drive nothing.
	//
	// The before-frame is captured first because AppModel.pages is a map: every
	// copy of the model shares the same component slice, so rendering the
	// pre-move model after the move renders the moved state and the comparison
	// passes vacuously.
	before := ansi.Strip(m.View().Content)
	moved := settle(t, m, tea.KeyPressMsg{Code: tea.KeyDown})

	if ansi.Strip(moved.View().Content) == before {
		t.Error("↓ changed nothing after returning to the page: the list never got its focus back")
	}
}

// The footer says what the arrows are doing right now. It is the one thing
// focus changes about this page's keys - restore is live on either half - so
// a bar that kept saying "navigate" over a focused preview would be lying
// about the only thing it had left to report.
func TestTheFooterSwapsNavigateForScrollOnFocus(t *testing.T) {
	m := onBackups(t)

	if hint := footerHints(m); !strings.Contains(hint, "navigate") {
		t.Errorf("the footer does not offer navigate with the list focused: %q", hint)
	}

	m = settle(t, m, tabKey)

	hint := footerHints(m)
	if !strings.Contains(hint, "scroll") {
		t.Errorf("the footer does not offer scroll with the preview focused: %q", hint)
	}
	if strings.Contains(hint, "navigate") {
		t.Errorf("the footer still offers navigate with the preview focused: %q", hint)
	}
}

// footerHints is the labels the bar would show, read off keys.Active with the
// model's own context - the same call the bar makes, so the two cannot
// disagree about what is pressable.
func footerHints(m AppModel) string {
	var labels []string
	for _, binding := range keys.Active(m.keyContext()) {
		labels = append(labels, binding.Help().Key+" "+binding.Help().Desc)
	}

	return strings.Join(labels, " · ")
}

// esc is not offered on this page: there is no selection to clear and no
// filter to abandon, so the key does nothing, and DESIGN.md's rule is that the
// bar does not advertise inert keys.
func TestTheBackupsFooterDoesNotOfferEsc(t *testing.T) {
	hint := footerHints(onBackups(t))

	if strings.Contains(hint, "back") {
		t.Errorf("the Backups footer still offers esc back: %q", hint)
	}
}

// Tab is advertised, because on this page alone it is live. The footer is
// where a user finds out the page has two halves at all.
func TestTheBackupsFooterOffersTab(t *testing.T) {
	if hint := footerHints(onBackups(t)); !strings.Contains(hint, "tab") {
		t.Errorf("the Backups footer does not mention tab: %q", hint)
	}

	// Negative control: it is not offered on a page with nothing to tab to.
	home := settle(t, withComposeLoaded(t),
		tea.WindowSizeMsg{Width: 120, Height: 40},
		cmds.SetActivePageMsg("Home"),
	)
	if hint := footerHints(home); strings.Contains(hint, "tab") {
		t.Errorf("Home's footer offers tab, which is dead there: %q", hint)
	}
}

// Restore fires from either half of the page. Focus moves the arrows, not the
// selection, and the selection is what a restore acts on.
func TestRestoreFiresWithThePreviewFocused(t *testing.T) {
	m := settle(t, onBackups(t), tabKey)

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	m = updated.(AppModel)

	for _, msg := range collect(cmd) {
		m = drive(m, msg)
	}

	if m.activeModal == nil {
		t.Error("r did not open the restore confirmation while the preview held focus")
	}
}

// enter is no longer a restore key anywhere on the page: too easy to hit by
// reflex while navigating, for an action that overwrites a live file.
func TestEnterDoesNotRestoreFromTheBackupsPage(t *testing.T) {
	if got := keys.Backup.Restore.Keys(); len(got) != 1 || got[0] != "r" {
		t.Fatalf("the restore binding carries %v, want r alone", got)
	}
	if key.Matches(tea.KeyPressMsg{Code: tea.KeyEnter}, keys.Backup.Restore) {
		t.Error("enter still matches the restore binding")
	}

	m := onBackups(t)
	m = settle(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})

	if m.activeModal != nil {
		t.Error("enter opened a modal on the Backups page; it should do nothing")
	}
}

// TestRigBackupsFocusEndToEnd drives the real program: open Backups, tab to
// the preview, tab back, and restore. It is the check the unit tests cannot
// make, because focus is spread across four packages that only meet inside a
// running program - AppModel matches the key, cmds carries the answer, both
// panels route by it and keys reports it. Every one of those seams is a place
// the pieces can each be right and still not add up.
func TestRigBackupsFocusEndToEnd(t *testing.T) {
	setupProjectDir(t)

	// A write through the app leaves a copy in the store, so the page has a
	// row to select and something to preview.
	if err := utils.SnapshotFile("compose.yaml"); err != nil {
		t.Fatalf("seeding a backup: %v", err)
	}

	r := newRig(t)
	if !r.WaitFor("core", 3*time.Second) {
		t.Fatal("the app never finished starting")
	}

	r.Send(letterKey('4'))
	if !r.WaitFor("Preview", 3*time.Second) {
		t.Fatalf("the Backups page never opened. Output:\n%s", r.Output())
	}

	// Wait for the store read to land before pressing anything. The page
	// renders - footer included - as soon as it is active, but the list
	// declines every key until it has entries, so a keystroke sent on the
	// strength of the footer alone races the read and is silently dropped.
	// The version count in the list's title is the read's own signal.
	if !r.WaitFor("1 version", 3*time.Second) {
		t.Fatalf("the version list never loaded. Screen:\n%s", r.scr.String())
	}

	// The footer is the page's own report of which half the arrows drive, and
	// it comes from keys.Active - the same call the handlers resolve through.
	if !r.WaitFor("navigate", 3*time.Second) {
		t.Fatalf("the page did not open on the list. Screen:\n%s", r.scr.String())
	}

	r.Send(keyPress(tea.KeyTab))
	if !r.WaitFor("scroll", 3*time.Second) {
		t.Fatalf("tab did not move focus to the preview. Screen:\n%s", r.scr.String())
	}
	if !r.WaitForNot("navigate", 3*time.Second) {
		t.Fatalf("the footer still offers navigate over the preview. Screen:\n%s", r.scr.String())
	}

	r.Send(keyPress(tea.KeyTab))
	if !r.WaitFor("navigate", 3*time.Second) {
		t.Fatalf("a second tab did not come back to the list. Screen:\n%s", r.scr.String())
	}

	// r restores from either half. Firing it after a round trip through focus
	// is the case worth driving: a restore wired to the focused panel rather
	// than to the selection would still pass every test that never tabs.
	//
	// This one is matched against the raw render stream rather than the
	// decoded screen. The confirm modal's frame makes the renderer emit a
	// delta the rig's decoder cannot place - it reconstructs a screen with the
	// modal's rows blank - so WaitFor reports the modal missing when the bytes
	// on the wire plainly contain it. The raw match is still sound here:
	// nothing earlier in this test renders that text, so it can only have come
	// from the confirmation opening. See screen_test_util.go's list of what the
	// decoder covers.
	r.Send(keyPress(tea.KeyTab))
	r.Send(letterKey('r'))

	if !waitForRaw(r, "Restore compose backup", 3*time.Second) {
		t.Fatalf("r did not open the restore confirmation from the preview. Screen:\n%s", r.scr.String())
	}
}

// waitForRaw polls the raw render stream, for the cases where what was drawn
// is not in question but where it landed on the reconstructed screen is.
func waitForRaw(r *rig, substr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if strings.Contains(r.Output(), substr) {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}

	return false
}

// The version list's filter is the same bubbles filter the groups and services
// lists have, so it has to claim the keyboard the same way: while one is being
// typed the page's verbs are letters, and while one stands esc clears it
// instead of doing nothing.
func TestFilteringTheBackupsListClaimsTheKeyboard(t *testing.T) {
	m := onBackups(t)

	if m.keyboardOwned() || m.escKept() {
		t.Fatal("precondition: an unfiltered Backups page claims neither the keyboard nor esc")
	}

	typing := settle(t, m, tea.KeyPressMsg{Code: '/', Text: "/"})
	if !typing.keyboardOwned() {
		t.Error("a filter being typed on Backups does not own the keyboard")
	}

	// r is a letter while the filter has the keyboard, not the restore verb.
	typed := settle(t, typing, tea.KeyPressMsg{Code: 'r', Text: "r"})
	if typed.activeModal != nil {
		t.Error("r opened the restore confirmation while it was being typed into the filter")
	}

}

// A standing filter hands esc to the list, and esc gets back out. That is the
// only thing esc does on this page - there is no selection to clear - so it is
// also why the footer offers the key here at all.
//
// A page of its own rather than a continuation of the test above: AppModel's
// pages map is shared by every copy of the model, so the list carries whatever
// the previous settles left in it. Filtering a list that is already mid-filter
// types "/" into the filter box and matches nothing.
func TestAnAppliedBackupsFilterKeepsEsc(t *testing.T) {
	// "compose" matches. "r" appears in no filter value here - the rows are
	// compose.yaml, a timestamp and a hex sha - and bubbles drops an accepted
	// filter with nothing under it straight back to unfiltered.
	applied := applyFilter(t, onBackups(t), "compose")

	if applied.keyboardOwned() {
		t.Error("an applied filter still owns the whole keyboard")
	}
	if !applied.escKept() {
		t.Fatal("an applied filter on Backups does not keep esc, so there is no way back to the full rows")
	}

	cleared := settle(t, applied, tea.KeyPressMsg{Code: tea.KeyEscape})
	if cleared.escKept() {
		t.Error("esc did not clear the filter, so the list is stuck narrowed")
	}
}

// applyFilter opens the filter, types term and accepts it.
//
// Each keystroke is settled before the next is sent, which is not a detail.
// bubbles narrows the rows through a command, so sending the whole sequence
// into one settle queues every key ahead of the first match: enter then lands
// on a list that has matched nothing yet, and bubbles drops an accepted filter
// with no matches straight back to unfiltered. The real runtime interleaves
// them, so the test has to as well.
func applyFilter(t *testing.T, m AppModel, term string) AppModel {
	t.Helper()

	m = settle(t, m, tea.KeyPressMsg{Code: '/', Text: "/"})
	for _, r := range term {
		m = settle(t, m, tea.KeyPressMsg{Code: r, Text: string(r)})
	}

	return settle(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
}

// The footer trades the filter key for the clear key once one stands, exactly
// as it does on Home and Services. esc is offered here and nowhere else on
// this page - there is no selection to clear - so this is also the check that
// the page stopped advertising an inert esc.
func TestTheBackupsFooterOffersTheFilterSlot(t *testing.T) {
	m := onBackups(t)

	hint := footerHints(m)
	if !strings.Contains(hint, "filter") {
		t.Errorf("the Backups footer does not offer the filter key: %q", hint)
	}
	if strings.Contains(hint, "back") {
		t.Errorf("the Backups footer offers esc back, which does nothing on that page: %q", hint)
	}

	filtered := applyFilter(t, m, "compose")

	if hint := footerHints(filtered); !strings.Contains(hint, "clear filter") {
		t.Errorf("the footer does not offer esc clear filter with one standing: %q", hint)
	}
}

// An empty store offers no filter key: there is nothing to narrow. AppModel
// cannot derive this from the config the way it does for Home and Services, so
// the panel answers for itself through listEmptier - this is the check that
// the wiring is actually connected.
func TestAnEmptyBackupStoreOffersNoFilter(t *testing.T) {
	// A project with no writes through the app has no stored versions.
	dir := t.TempDir()
	compose := dir + "/compose.yaml"
	writeFile(t, compose, "services:\n  app:\n    image: nginx:alpine\n")

	m := startup(120, 40)
	updated, cmd := m.Update(cmds.GetConfigMsg{
		FileName: compose,
		Files:    []string{compose},
		Project:  groupProject(),
	})
	m = settle(t, drive(updated, collect(cmd)...), cmds.SetActivePageMsg("Backups"))

	if hint := footerHints(m); strings.Contains(hint, "filter") {
		t.Errorf("an empty backup store still offers the filter key: %q", hint)
	}
}
