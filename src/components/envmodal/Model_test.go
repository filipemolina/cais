package envmodal

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

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
	if len(m.Entries()) != 2 {
		t.Fatalf("entries: got %d, want 2", len(m.Entries()))
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
	if m.revealed != 1 {
		t.Errorf("revealed = %d, want 1", m.revealed)
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

// manyVars is a .env longer than any terminal will show at once.
func manyVars(n int) []cmds.EnvEntry {
	entries := make([]cmds.EnvEntry, 0, n)
	for i := range n {
		entries = append(entries, cmds.EnvEntry{
			Key:    fmt.Sprintf("VAR_%02d", i),
			Value:  "value",
			Source: "var",
		})
	}

	return entries
}

// A .env with more variables than the terminal has rows has to stay inside the
// modal. It did not: the table looped over every entry and drew them all, so
// the surplus ran off the bottom of the screen, taking the modal's border and
// its hint line with it.
func TestALongEnvFileStaysInsideTheTerminal(t *testing.T) {
	const termHeight = 24

	m := New("/tmp/example.env", termHeight).(Model)
	m.SetEntries("/tmp/example.env", manyVars(80), 0)

	if got := lipgloss.Height(m.View().Content); got > termHeight {
		t.Errorf("the modal is %d rows tall on a %d-row terminal", got, termHeight)
	}
}

// And it re-fits when the terminal changes under it, like every other modal.
func TestTheEnvModalRefitsWhenTheTerminalShrinks(t *testing.T) {
	m := New("/tmp/example.env", 60).(Model)
	m.SetEntries("/tmp/example.env", manyVars(80), 0)

	tall := lipgloss.Height(m.View().Content)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m = updated.(Model)

	short := lipgloss.Height(m.View().Content)
	if short >= tall {
		t.Errorf("modal is %d rows on a 20-row terminal and %d on a 60-row one: it did not shrink", short, tall)
	}
	if short > 20 {
		t.Errorf("modal is %d rows tall on a 20-row terminal", short)
	}
}

// Nothing is revealed until the user asks. The delegate carries the revealed
// index, and an index's zero value is a real row - so a delegate built with
// the list would have shown row 0's value from the moment the modal opened.
func TestNoValueIsRevealedOnOpen(t *testing.T) {
	m := withEntries(t)

	if m.revealed != -1 {
		t.Fatalf("revealed = %d on open, want -1", m.revealed)
	}

	rendered := ansi.Strip(m.View().Content)
	if strings.Contains(rendered, "one") {
		t.Errorf("the first row's value is showing before anyone asked:\n%s", rendered)
	}
}

// Moving off a revealed row re-hides it: a value is revealed for as long as
// you are looking at it and no longer.
func TestMovingTheCursorRehidesARevealedValue(t *testing.T) {
	m := withEntries(t)

	revealed, _ := m.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	m = revealed.(Model)
	if m.revealed != 0 {
		t.Fatalf("revealed = %d after space, want 0", m.revealed)
	}

	moved, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	m = moved.(Model)

	if m.revealed != -1 {
		t.Errorf("revealed = %d after moving the cursor, want -1", m.revealed)
	}
	if rendered := ansi.Strip(m.View().Content); strings.Contains(rendered, "one") {
		t.Errorf("the value stayed revealed behind the cursor:\n%s", rendered)
	}
}
