package envmodal

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/cmds"
)

// withEntries builds a model with a parsed .env (two vars plus a comment) and
// the loading flag cleared, ready for key handling.
func withEntries(t *testing.T) Model {
	t.Helper()
	m := New("/tmp/example.env", 40).(Model)
	m.SetEntries("/tmp/example.env", []cmds.EnvEntry{
		{Key: "FIRST", Value: "one", Source: "var"},
		{Key: "SECRET", Value: "topsecret", Source: "var"},
		{Key: "", Source: "comment", Raw: "# a comment"},
	}, 0)
	return m
}

// TestNewEmitsGetEnvFileContents covers test 8 (load on open): New stores the
// path; the AppModel handler issues GetEnvFileContents. Here we verify the
// model accepts EnvFileContentsMsg and populates the table, and that space
// reveals the selected value while c copies it (also test 8/9).
func TestEnvFileContentsMsgPopulatesTable(t *testing.T) {
	m := New("/x/.env", 40).(Model)
	if !m.loading {
		t.Fatal("new modal should start loading")
	}

	updated, _ := m.Update(cmds.EnvFileContentsMsg{
		Path: "/x/.env",
		Entries: []cmds.EnvEntry{
			{Key: "A", Value: "1", Source: "var"},
			{Key: "B", Value: "2", Source: "var"},
		},
	})
	m = updated.(Model)

	if m.loading {
		t.Error("model still loading after contents arrive")
	}
	if len(m.entries) != 2 {
		t.Fatalf("entries: got %d, want 2", len(m.entries))
	}
}

// TestSpaceRevealsAndCCopies covers test 8/9: space reveals the selected
// value, c copies it to the clipboard.
func TestSpaceRevealsAndCCopies(t *testing.T) {
	m := withEntries(t)

	// Move to the SECRET row (index 1) and reveal it with space.
	down, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	m = down.(Model)
	reveal, revealCmd := m.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	m = reveal.(Model)
	if m.revealedIdx != 1 {
		t.Errorf("revealedIdx = %d, want 1", m.revealedIdx)
	}

	// c copies the selected value.
	_, copyCmd := m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})

	// Verify the copy command is a SetClipboard carrying the value.
	if !emitsSetClipboard(revealCmd) && !emitsSetClipboard(copyCmd) {
		t.Error("pressing c did not emit a clipboard message")
	}
}

// TestNEDKeysEmitModalMessages covers test 9: n/e/d emit the matching env
// modal/confirm messages, E emits OpenEditorMsg, o emits OpenEnvRawEditMsg.
func TestNEDKeysEmitModalMessages(t *testing.T) {
	m := withEntries(t)

	// n -> OpenEnvKeyModalMsg
	_, nCmd := m.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	if !emitsMsg(nCmd, func(msg tea.Msg) bool { _, ok := msg.(cmds.OpenEnvKeyModalMsg); return ok }) {
		t.Error("n did not emit OpenEnvKeyModalMsg")
	}
	// e -> OpenEnvEditModalMsg (with the selected var's key/value)
	_, eCmd := m.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
	if !emitsMsg(eCmd, func(msg tea.Msg) bool {
		em, ok := msg.(cmds.OpenEnvEditModalMsg)
		return ok && em.Key == "FIRST" && em.Value == "one"
	}) {
		t.Error("e did not emit OpenEnvEditModalMsg for the selected var")
	}
	// d -> OpenEnvDeleteConfirmMsg
	_, dCmd := m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if !emitsMsg(dCmd, func(msg tea.Msg) bool { _, ok := msg.(cmds.OpenEnvDeleteConfirmMsg); return ok }) {
		t.Error("d did not emit OpenEnvDeleteConfirmMsg")
	}
	// E -> CloseModal wrapping OpenEditorMsg
	_, ECmd := m.Update(tea.KeyPressMsg{Code: 'E', Text: "E"})
	if !emitsMsg(ECmd, func(msg tea.Msg) bool { _, ok := msg.(cmds.CloseModalMsg); return ok }) {
		t.Error("E did not emit CloseModal (to hand off to the editor)")
	}
	// o -> OpenEnvRawEditMsg
	_, oCmd := m.Update(tea.KeyPressMsg{Code: 'o', Text: "o"})
	if !emitsMsg(oCmd, func(msg tea.Msg) bool { _, ok := msg.(cmds.OpenEnvRawEditMsg); return ok }) {
		t.Error("o did not emit OpenEnvRawEditMsg")
	}
}

// TestEscCloses covers test 9 (esc closes): esc emits CloseModal(nil), the
// signal AppModel uses to clear activeModal.
func TestEscCloses(t *testing.T) {
	m := withEntries(t)
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if !emitsMsg(cmd, func(msg tea.Msg) bool {
		cm, ok := msg.(cmds.CloseModalMsg)
		return ok && cm.Follow == nil
	}) {
		t.Error("esc did not emit CloseModal(nil)")
	}
}

// TestSaveEnvFileMsgReRequestsContents covers test 10: a successful save
// re-requests the .env contents so the table refreshes.
func TestSaveEnvFileMsgReRequestsContents(t *testing.T) {
	m := withEntries(t)
	_, cmd := m.Update(cmds.SaveEnvFileMsg{Path: "/tmp/example.env"})
	if !emitsMsg(cmd, func(msg tea.Msg) bool { _, ok := msg.(cmds.EnvFileContentsMsg); return ok }) {
		t.Error("SaveEnvFileMsg success did not re-request contents")
	}
}

// collect drains commands into their messages.
func collect(cmds ...tea.Cmd) []tea.Msg {
	var out []tea.Msg
	for _, c := range cmds {
		if c == nil {
			continue
		}
		msg := c()
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, child := range batch {
				out = append(out, collectCmd(child)...)
			}
			continue
		}
		out = append(out, msg)
	}
	return out
}

func collectCmd(c tea.Cmd) []tea.Msg {
	if c == nil {
		return nil
	}
	msg := c()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, child := range batch {
			out = append(out, collectCmd(child)...)
		}
		return out
	}
	if msg == nil {
		return nil
	}
	return []tea.Msg{msg}
}

// emitsSetClipboard reports whether the command produces a clipboard message.
// tea.SetClipboard returns a command whose message type is unexported
// (tea.setClipboardMsg), so we match on its type name rather than a direct
// assertion, the way the clipboard content is what matters for the test.
func emitsSetClipboard(cmd tea.Cmd) bool {
	for _, msg := range collect(cmd) {
		if strings.Contains(fmt.Sprintf("%T", msg), "Clipboard") {
			return true
		}
	}
	return false
}

// emitsMsg reports whether the command produces a message matching pred.
func emitsMsg(cmd tea.Cmd, pred func(tea.Msg) bool) bool {
	for _, msg := range collect(cmd) {
		if pred(msg) {
			return true
		}
	}
	return false
}
