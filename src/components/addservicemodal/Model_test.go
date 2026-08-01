package addservicemodal

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/cmds"
)

func specialKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

func letterKey(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

// followRequest drains a CloseModalMsg's follow command and returns the
// message it produces, failing if the modal did not close with one.
func followRequest(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()

	if cmd == nil {
		t.Fatal("submit produced no command")
	}
	msg := cmd()
	closeMsg, ok := msg.(cmds.CloseModalMsg)
	if !ok {
		t.Fatalf("expected CloseModalMsg, got %T", msg)
	}
	if closeMsg.Follow == nil {
		t.Fatal("modal closed without a follow command")
	}

	return closeMsg.Follow()
}
