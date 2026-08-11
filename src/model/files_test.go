package model

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/filipemolina/cais/src/cmds"
)

// Switching to the Files page asks for the compose file's contents, so the
// viewport is never stale from having been inactive during a write.
func TestSwitchingToTheFilesPageReadsTheComposeFile(t *testing.T) {
	m := withGroupsLoaded(t)

	if got, want := m.config.configFileName, "compose.yaml"; got != want {
		t.Fatalf("precondition: loaded file = %q, want %q", got, want)
	}

	_, cmd := m.Update(cmds.SetActivePageMsg("Compose Files"))

	var reads bool
	for _, c := range flattenCmds(cmd) {
		if msg := c(); msg != nil {
			if _, ok := msg.(cmds.ComposeFileContentsMsg); ok {
				reads = true
			}
		}
	}
	if !reads {
		t.Error("switching to the Files page did not read the compose file")
	}
}

// Staying on Home does not read the file for the Files page.
func TestStayingOffTheFilesPageSkipsTheRead(t *testing.T) {
	m := withGroupsLoaded(t)

	if cmd := m.recomposeFilesCmdIfActive(); cmd != nil {
		t.Error("recomposeFilesCmdIfActive returned a command while on Home")
	}
}

// The Files page renders the file's path and its contents once both arrive.
func TestFilesPageRendersPathAndContents(t *testing.T) {
	m := withGroupsLoaded(t)

	m = drive(m,
		cmds.SetActivePageMsg("Compose Files"),
		cmds.SetComposeFileMsg{Name: "compose.yaml"},
		cmds.ComposeFileContentsMsg{
			Name:     "compose.yaml",
			Contents: "services:\n  web:\n    image: nginx:alpine\n",
		},
	)
	m = applyLayout(m)

	frame := ansi.Strip(m.View().Content)

	if !strings.Contains(frame, "compose.yaml") {
		t.Errorf("Files page does not name the compose file:\n%s", frame)
	}
	if !strings.Contains(frame, "nginx:alpine") {
		t.Errorf("Files page does not show the file's contents:\n%s", frame)
	}
}

// A write through the app while the Files page is showing re-reads the
// file, so the viewport reflects the change on the same frame the config
// reload does.
func TestAWriteWhileOnTheFilesPageReReadsIt(t *testing.T) {
	m := withGroupsLoaded(t)
	m = drive(m, cmds.SetActivePageMsg("Compose Files"))

	_, cmd := m.Update(cmds.CreateGroupMsg{})

	var reads bool
	for _, c := range flattenCmds(cmd) {
		if msg := c(); msg != nil {
			if _, ok := msg.(cmds.ComposeFileContentsMsg); ok {
				reads = true
			}
		}
	}
	if !reads {
		t.Error("a write while the Files page was showing did not re-read the file")
	}
}

// flattenCmds unwraps a tea.Batch into its member commands so a test can
// run each and inspect the messages. Unlike collect, this keeps the
// commands rather than the messages, which matters for ComposeFileContentsMsg:
// it is produced by running GetComposeFileContents, not carried as data.
func flattenCmds(cmd tea.Cmd) []tea.Cmd {
	if cmd == nil {
		return nil
	}

	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		return batch
	}
	if msg == nil {
		return nil
	}

	// A single non-batch command: re-wrap it so the caller can invoke it.
	return []tea.Cmd{func() tea.Msg { return msg }}
}
