package model

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/confirmmodal"
	"github.com/filipemolina/cais/src/components/envmodal"
	"github.com/filipemolina/cais/src/components/errormodal"
	"github.com/filipemolina/cais/src/utils"
)

// withComposeLoaded is the app on Home with a compose file loaded, laid out,
// and a .env known next to it. A real backup of compose.yaml is seeded so
// restore flows have a copy to act on.
func withComposeLoaded(t *testing.T) AppModel {
	t.Helper()

	dir := t.TempDir()
	compose := dir + "/compose.yaml"
	env := dir + "/.env"
	writeFile(t, compose, "services:\n  app:\n    image: nginx:alpine\n")
	writeFile(t, env, "SECRET=1\n")

	m := startup(120, 40)
	updated, cmd := m.Update(cmds.GetConfigMsg{
		FileName: compose,
		Files:    []string{compose},
		Project:  groupProject(),
	})
	m = drive(updated, collect(cmd)...)
	// Seed the envPath the config sync would set (GetConfigMsg sets it from
	// the file name's directory).
	m.config.envPath = env

	// Seed one real backup of the compose file.
	if err := utils.SnapshotFile(compose); err != nil {
		t.Fatalf("seeding backup: %v", err)
	}

	return applyLayout(m)
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// TestSwitchingToBackupsPageReadsBackups covers test 11: activating the
// Backups page issues GetBackups so the list fills.
func TestSwitchingToBackupsPageReadsBackups(t *testing.T) {
	m := withComposeLoaded(t)

	_, cmd := m.Update(cmds.SetActivePageMsg("Backups"))

	var reads bool
	for _, c := range flattenCmds(cmd) {
		if msg := c(); msg != nil {
			if _, ok := msg.(cmds.BackupListMsg); ok {
				reads = true
			}
		}
	}
	if !reads {
		t.Error("switching to the Backups page did not issue GetBackups (BackupListMsg)")
	}
}

// TestEnvIsNoLongerAPage covers test 11: "Env" is not a page, so activating it
// is a no-op that does not read the (former) env file.
func TestEnvIsNoLongerAPage(t *testing.T) {
	m := withComposeLoaded(t)

	_, cmd := m.Update(cmds.SetActivePageMsg("Env"))

	for _, c := range flattenCmds(cmd) {
		if msg := c(); msg != nil {
			if _, ok := msg.(cmds.EnvFileContentsMsg); ok {
				t.Error("activating \"Env\" still issued an env read; the page was removed")
			}
		}
	}
}

// TestGlobalVOpensEnvModal covers test 11: pressing v with a loaded envPath
// opens the env modal; with no envPath it is a no-op.
func TestGlobalVOpensEnvModal(t *testing.T) {
	m := withComposeLoaded(t)

	// v emits OpenEnvModalMsg, which the model handles on the next cycle to
	// open the modal and load its contents.
	updated, vCmd := m.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	m = updated.(AppModel)
	for _, msg := range collect(vCmd) {
		if oem, ok := msg.(cmds.OpenEnvModalMsg); ok {
			updated2, _ := m.Update(oem)
			m = updated2.(AppModel)
		}
	}

	if m.activeModal == nil {
		t.Fatal("v did not open a modal")
	}
	if _, ok := m.activeModal.(envmodal.Model); !ok {
		t.Fatalf("v opened %T, want an envmodal.Model", m.activeModal)
	}

	// With no envPath, v is a no-op (no modal, no error).
	m2 := withComposeLoaded(t)
	m2.config.envPath = ""
	updated2, _ := m2.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	m2 = updated2.(AppModel)
	if m2.activeModal != nil {
		t.Error("v opened a modal when envPath was empty (should be a no-op)")
	}
}

// seedComposeBackupName returns the name of the single seeded compose backup,
// or fails the test if none exists.
func seedComposeBackupName(t *testing.T, m AppModel) string {
	t.Helper()
	backups, err := utils.ListBackups(m.config.configFileName)
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(backups) == 0 {
		t.Fatal("expected a seeded compose backup")
	}
	return backups[0].Name
}

// TestRequestRestoreBackupOpensConfirm covers test 12: pressing r on a backup
// row (via RequestRestoreBackupMsg) opens a confirm modal whose confirm action
// is the restore command.
func TestRequestRestoreBackupOpensConfirm(t *testing.T) {
	m := withComposeLoaded(t)
	name := seedComposeBackupName(t, m)

	updated, _ := m.Update(cmds.RequestRestoreBackupMsg{Source: "compose", Name: name})
	m = updated.(AppModel)

	if m.activeModal == nil {
		t.Fatal("RequestRestoreBackupMsg did not open a modal")
	}
	if _, ok := m.activeModal.(confirmmodal.Model); !ok {
		t.Fatalf("RequestRestoreBackupMsg opened %T, want a confirmmodal.Model", m.activeModal)
	}
}

// TestConfirmingRestoreRunsRestoreCommand covers test 12: confirming the
// restore runs the RestoreBackup command; a successful RestoreBackupMsg clears
// the error, reloads the config, and re-lists the backups.
func TestConfirmingRestoreRunsRestoreCommand(t *testing.T) {
	m := withComposeLoaded(t)
	name := seedComposeBackupName(t, m)

	// Be on the Backups page so a successful restore re-lists the store.
	m = drive(m, cmds.SetActivePageMsg("Backups"))

	// Open the confirm modal and confirm with y.
	m = drive(m, cmds.RequestRestoreBackupMsg{Source: "compose", Name: name})
	updated, confirmCmd := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	m = updated.(AppModel)

	// The confirm's follow is a RestoreBackup command. Drive it to a
	// RestoreBackupMsg and feed that back through the model.
	var restoreMsg tea.Msg
	for _, msg := range collect(confirmCmd) {
		if closeMsg, ok := msg.(cmds.CloseModalMsg); ok && closeMsg.Follow != nil {
			restoreMsg = closeMsg.Follow()
		}
	}
	if restoreMsg == nil {
		t.Fatal("confirming restore did not run the restore command")
	}
	rbm, ok := restoreMsg.(cmds.RestoreBackupMsg)
	if !ok {
		t.Fatalf("restore follow produced %T, want cmds.RestoreBackupMsg", restoreMsg)
	}
	if rbm.Err != nil {
		t.Fatalf("restore command failed on a real backup: %v", rbm.Err)
	}

	// Success path: a RestoreBackupMsg with no error reloads and re-lists.
	updated, successCmd := m.Update(rbm)
	m = updated.(AppModel)

	if m.lastError != "" {
		t.Errorf("success left an error: %q", m.lastError)
	}
	var reloaded, relisted bool
	for _, c := range flattenCmds(successCmd) {
		if msg := c(); msg != nil {
			switch msg.(type) {
			case cmds.GetConfigMsg:
				reloaded = true
			case cmds.BackupListMsg:
				relisted = true
			}
		}
	}
	if !reloaded {
		t.Error("a successful restore did not reload the config")
	}
	if !relisted {
		t.Error("a successful restore did not re-list the backups")
	}
}

// TestRestoreErrorReportsForegroundError covers test 12: a failed restore
// reports the foreground error (an error modal when none is open, the banner
// when one is).
func TestRestoreErrorReportsForegroundError(t *testing.T) {
	m := withComposeLoaded(t)

	updated, _ := m.Update(cmds.RestoreBackupMsg{Err: errBoom{}})
	m = updated.(AppModel)

	if m.lastError == "" && m.activeModal == nil {
		t.Error("a failed restore left no foreground error and no error modal")
	}
	if m.activeModal != nil {
		if _, ok := m.activeModal.(errormodal.Model); !ok {
			t.Errorf("failed restore opened %T, want an errormodal.Model", m.activeModal)
		}
	}
}

// TestBackupsPageRendersTitle covers a smoke check that the Backups page
// renders its title without panicking, even with no backups yet.
func TestBackupsPageRendersTitle(t *testing.T) {
	m := withComposeLoaded(t)
	m = drive(m, cmds.SetActivePageMsg("Backups"))

	frame := ansi.Strip(m.View().Content)
	if !strings.Contains(frame, "Backups") {
		t.Errorf("Backups page does not render its title:\n%s", frame)
	}
}

// The Backups page is two panels, not one. Before the split it was a single
// component that drew both columns inside one frame, which meant the body
// gutter, the per-panel sizing and the draggable split all had to be
// reimplemented inside it. Rendering both titles is the cheapest proof that
// AppModel.pages holds two entries and renderBody laid them out.
//
// The window size is not optional here: without it the panels are zero-width
// and every assertion about their contents passes vacuously.
func TestBackupsPageRendersTwoPanels(t *testing.T) {
	m := withComposeLoaded(t)
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m = drive(m, cmds.SetActivePageMsg("Backups"))

	frame := ansi.Strip(m.View().Content)
	for _, title := range []string{"Backups", "Preview"} {
		if !strings.Contains(frame, title) {
			t.Errorf("the Backups page does not render the %q panel:\n%s", title, frame)
		}
	}
}

// settle runs the model forward the way the Bubble Tea runtime does: execute
// whatever a message produced, feed the results back, and repeat until the
// model stops asking for anything.
//
// drive alone is not enough for the Backups page. Activating it issues
// GetBackups, whose BackupListMsg makes the list publish a selection, whose
// SetSelectedBackupMsg makes the preview issue a read - a four-message chain
// where drive stops after the first. The iteration cap keeps a repeating
// command (a poll, a spinner tick) from spinning here forever.
func settle(t *testing.T, m AppModel, msgs ...tea.Msg) AppModel {
	t.Helper()

	queue := append([]tea.Msg{}, msgs...)

	for range 50 {
		if len(queue) == 0 {
			return m
		}

		msg := queue[0]
		queue = queue[1:]

		updated, cmd := m.Update(msg)
		m = updated.(AppModel)
		queue = append(queue, collect(cmd)...)
	}

	t.Fatal("the model never settled: a command kept producing messages")
	return m
}

// Selecting a version in the list reaches the preview panel. The two are
// siblings under AppModel with no reference to each other, so the only thing
// holding the page together is that SetSelectedBackupMsg is broadcast to the
// active page's components. If the list stops publishing, the preview sits
// on its empty state forever and nothing else fails.
func TestSelectingABackupReachesThePreview(t *testing.T) {
	m := withComposeLoaded(t)

	// A write to the compose file leaves a copy in the store to select.
	if err := utils.SnapshotFile(m.config.configFileName); err != nil {
		t.Fatalf("seeding a backup: %v", err)
	}

	m = settle(t, m,
		tea.WindowSizeMsg{Width: 120, Height: 40},
		cmds.SetActivePageMsg("Backups"),
	)

	frame := ansi.Strip(m.View().Content)

	// The preview names the copy it is showing as "<source> · <sha8>" in its
	// title. The list writes the same sha as "sha8 <sha8>", so the separator
	// is what makes this the preview's line and not a row of the list.
	if !strings.Contains(frame, "compose · ") {
		t.Errorf("the preview never received a selection from the list:\n%s", frame)
	}
	if strings.Contains(frame, "Nothing selected") {
		t.Errorf("the preview is still showing its empty state:\n%s", frame)
	}
}
