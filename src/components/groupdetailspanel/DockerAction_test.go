package groupdetailspanel

import (
	"testing"

	"github.com/filipemolina/cais/src/cmds"

	tea "charm.land/bubbletea/v2"
)

// keyPress builds the KeyPressMsg for a single rune, the way a panel sees it.
// Its own copy: DockerAction_test.go's version was shared for free with
// DetailsPanel's half of this test while both lived in the flat components
// package, and no longer is.
func keyPress(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

// requestedAction runs cmd and asserts it produced an intent for AppModel
// rather than a finished docker call.
func requestedAction(t *testing.T, cmd tea.Cmd) cmds.RunDockerActionMsg {
	t.Helper()

	if cmd == nil {
		t.Fatal("expected a command, got nil")
	}

	msg := cmd()
	request, ok := msg.(cmds.RunDockerActionMsg)
	if !ok {
		t.Fatalf("expected a RunDockerActionMsg, got %T", msg)
	}

	return request
}

// The panel does not know which compose file is loaded and must not run docker
// itself: it asks, and AppModel - which does know - runs it with --file. If
// this ever goes back to returning a DockerActionMsg, the action has run
// against whatever file docker resolved on its own, which is the desync the
// --file threading exists to prevent.
func TestGroupDetailsPanelRequestsTheActionRatherThanRunningIt(t *testing.T) {
	panel := New()
	panel, _ = panel.Update(cmds.SetSelectedGroupMsg("backend"))

	_, cmd := panel.Update(keyPress('t'))

	request := requestedAction(t, cmd)
	if request.Action != "stop" || request.Target != "backend" || !request.IsGroup {
		t.Errorf("request: got %+v, want stop/backend/group", request)
	}
}
