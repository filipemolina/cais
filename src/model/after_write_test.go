package model

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/cmds"
)

// filesPageWithProject is the app on the Files page with a compose file
// loaded - the only state in which recomposeFilesCmdIfActive produces
// anything.
func filesPageWithProject(t *testing.T) AppModel {
	t.Helper()

	m := startup(120, 40)
	updated, cmd := m.Update(cmds.SetActivePageMsg(apptypes.PageComposeFiles))
	m = drive(updated, collect(cmd)...)
	updated, cmd = m.Update(cmds.GetConfigMsg{FileName: "compose.yaml", Project: groupProject()})

	return applyLayout(drive(updated, collect(cmd)...))
}

func rereadsTheFile(msgs []tea.Msg) bool {
	for _, msg := range msgs {
		if _, ok := msg.(cmds.ComposeFileContentsMsg); ok {
			return true
		}
	}

	return false
}

// Every compose-file write re-reads the file for the Files page when that is
// the page on screen, because the page is showing the file that just changed.
//
// EditGroupMsg used to skip it, and CycleRestartPolicyMsg used to skip the
// layout rebroadcast instead - mirror images, both oversights rather than
// rules. afterWrite gives every write both, and this pins the first half
// across all of them so a future write cannot quietly rejoin the exception.
func TestEveryWriteRefreshesTheFilesPage(t *testing.T) {
	writes := map[string]tea.Msg{
		"edit group":           cmds.EditGroupMsg{},
		"cycle restart policy": cmds.CycleRestartPolicyMsg{},
		"rename group":         cmds.RenameGroupMsg{NewName: "core2"},
		"create group":         cmds.CreateGroupMsg{},
		"delete group":         cmds.DeleteGroupMsg{},
		"adopt ungrouped":      cmds.AdoptUngroupedMsg{},
		"release ungrouped":    cmds.ReleaseUngroupedMsg{},
		"delete service":       cmds.DeleteServiceMsg{},
		"add healthcheck":      cmds.AddHealthcheckMsg{},
	}

	for name, msg := range writes {
		t.Run(name, func(t *testing.T) {
			m := filesPageWithProject(t)

			_, cmd := m.Update(msg)

			if !rereadsTheFile(collect(cmd)) {
				t.Error("a successful write did not re-read the compose file for the Files page")
			}
		})
	}
}

// The other half: a write that clears the error banner gives back the screen
// row the banner was costing, and the panels have to be told. This is what
// CycleRestartPolicyMsg was missing.
func TestAWriteThatClearsTheBannerRebroadcastsTheLayout(t *testing.T) {
	m := withGroupsLoaded(t)

	// Put a banner up and settle the stored layout around it, the way the
	// poll's own error path does.
	m.lastError = "docker daemon unavailable"
	m.lastErrorFromPoll = true
	m.rebroadcastBodyLayoutIfChanged()
	withBanner := m.config.bodyLayout

	updated, cmd := m.Update(cmds.CycleRestartPolicyMsg{})
	m = updated.(AppModel)

	if m.lastError != "" {
		t.Fatalf("a successful write left the banner up: %q", m.lastError)
	}
	if m.config.bodyLayout == withBanner {
		t.Fatal("the layout did not change when the banner went, so there is nothing to rebroadcast")
	}

	var told bool
	for _, msg := range collect(cmd) {
		if _, ok := msg.(cmds.SetBodyLayoutMsg); ok {
			told = true
		}
	}
	if !told {
		t.Error("the panels were not told the banner's row is theirs again")
	}
}
