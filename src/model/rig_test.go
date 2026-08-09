package model

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/filipemolina/stack-stitcher/src/utils"
)

// rig drives a real tea.Program end-to-end without a TTY. It captures all
// rendered output to a buffer and lets the test inject messages via Send().
// Use the helpers (Send, Latest, WaitFor) to drive it.
//
// The rig uses Bubble Tea's default renderer (no WithoutRenderer) so the
// full render pipeline runs, but redirects the output to an in-memory buffer
// instead of a real terminal. The output is therefore a stream of ANSI
// escape sequences interleaved with rendered text; substring matching is the
// primary assertion style.
type rig struct {
	p      *tea.Program
	out    *safeBuffer
	scr    *screen
	cursor int
	done   chan struct{}
}

type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func newRig(t *testing.T) *rig {
	t.Helper()

	out := &safeBuffer{}
	scr := newScreen(120, 40)
	// tee feeds the raw stream to the screen decoder as well as the buffer.
	tee := &teeWriter{out: out, scr: scr}
	p := tea.NewProgram(
		GetInitialModel(utils.ComposeSource{}),
		tea.WithInput(nil),
		tea.WithOutput(tee),
		tea.WithoutSignals(),
		tea.WithWindowSize(120, 40),
	)

	r := &rig{p: p, out: out, scr: scr, done: make(chan struct{})}
	go func() {
		defer close(r.done)
		_, _ = p.Run()
	}()

	t.Cleanup(func() {
		p.Quit()
		<-r.done
	})

	return r
}

// teeWriter fans each write out to both the raw buffer and the screen
// decoder, so the two views of the render stream stay in lockstep.
type teeWriter struct {
	out *safeBuffer
	scr *screen
}

func (w *teeWriter) Write(p []byte) (int, error) {
	w.scr.feed(string(p))
	return w.out.Write(p)
}

// Send injects a message into the program loop, the same way the
// tea.NewProgram API exposes for testing.
func (r *rig) Send(msg tea.Msg) {
	r.p.Send(msg)
}

// letterKey builds a KeyPressMsg for a printable character key, with both
// Code and Text set. Both matter for their own reason: key.Matches (used by
// the panel handlers) compares msg.String() against the binding strings, so
// a letter sent with only Code does not match a panel binding; textinput
// (charm.land/bubbles/v2/textinput.Model.Update) inserts msg.Text, not
// msg.Code, so a letter with only Code types nothing at all. Use this
// helper for any rig key meant to reach a panel binding (s/t/r/p/x/l/e/n/d)
// or type into a field; keep keyPress for special keys (esc, enter, tab,
// backspace), where Code alone is what both textinput and key.Matches key
// off of.
func letterKey(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

// Latest returns the bytes rendered since the last call to Latest (or since
// the rig was created). It is safe to call from the test goroutine while
// the program goroutine is concurrently writing to the buffer.
func (r *rig) Latest() string {
	full := r.out.String()
	delta := full[r.cursor:]
	r.cursor = len(full)
	return delta
}

// WaitFor polls the decoded current screen until substr is visible or the
// timeout elapses. Returns true if found. Because it matches the current
// screen rather than the append-only byte history, it never reports a
// substring that was rendered and then overwritten (no false positives), and
// it never misses a substring because a previous call consumed the bytes (no
// false negatives).
func (r *rig) WaitFor(substr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(r.scr.String(), substr) {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

// WaitForNot polls the decoded current screen until substr is no longer
// visible, or the timeout elapses. Returns true if the substring disappeared
// within the timeout. Useful for asserting that a modal or banner has been
// dismissed: the match is against what is currently on screen, so a modal
// closed in an earlier frame does not linger as a ghost.
func (r *rig) WaitForNot(substr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !strings.Contains(r.scr.String(), substr) {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

// Output returns the full captured output since the rig was created.
// Useful for debugging test failures.
func (r *rig) Output() string {
	return r.out.String()
}

// TestRigGroupListEditKey reaches the focused groups list through the rig:
// 'e' opens the membership editor. This used to fail because panel key
// bindings match on msg.String(), which comes from Text, not Code.
func TestRigGroupListEditKey(t *testing.T) {
	setupProjectDir(t)

	r := newRig(t)
	if !r.WaitFor("core", 3*time.Second) {
		t.Fatal("groups never rendered")
	}

	r.Send(letterKey('e'))

	if !r.WaitFor("Edit members of", 3*time.Second) {
		t.Fatalf("expected membership editor modal. Output:\n%s", r.Output())
	}
}

// TestRigRenameGroup drives the rename flow end to end: R opens the prompt,
// typing a new name and Enter write the compose file through the real
// reload, and the re-derived list shows the renamed group.
func TestRigRenameGroup(t *testing.T) {
	setupProjectDir(t)

	r := newRig(t)
	if !r.WaitFor("core", 3*time.Second) {
		t.Fatal("groups never rendered")
	}

	r.Send(letterKey('R'))

	if !r.WaitFor("Rename group", 3*time.Second) {
		t.Fatalf("rename prompt never opened. Output:\n%s", r.Output())
	}

	// The input is pre-filled with "core"; typing appends at the cursor.
	r.Send(letterKey('2'))
	r.Send(keyPress(tea.KeyEnter))

	// The modal closes, then the reloaded list shows the new name. Wait for
	// the modal to go first so "core2" cannot be matched inside its input.
	if !r.WaitForNot("Rename group", 3*time.Second) {
		t.Fatalf("rename modal did not close. Output:\n%s\n=== DECODED ===\n%s", r.Output(), r.scr.String())
	}
	if !r.WaitFor("core2", 3*time.Second) {
		t.Fatalf("renamed group never appeared in the list. Output:\n%s", r.Output())
	}
}

// TestRigAddService drives the add-service flow end to end: n on the
// Services page opens the two-field modal, typing a name and image and
// pressing Enter lands the service in the compose file and opens the inline
// editor on it. That last step is the race worth checking - AddServiceMsg's
// handler batches a reload, a focus change and the inline-edit request
// together, and Bubble Tea's Batch makes no ordering promises between them,
// so this exercises the real timing rather than asserting it in isolation.
func TestRigAddService(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(panelKeyFixture), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	t.Chdir(dir)

	r := newRig(t)
	if !r.WaitFor("web", 3*time.Second) {
		t.Fatalf("groups never rendered. Output:\n%s", r.Output())
	}

	r.Send(keyPress('2')) // Services page
	// "PROPERTY" is the config table header the details panel renders on the
	// Services page only; the tab bar label "Services" is on screen on every
	// page, so waiting on it would race the page switch.
	if !r.WaitFor("PROPERTY", 3*time.Second) {
		t.Fatalf("did not switch to the Services page. Output:\n%s", r.Output())
	}

	r.Send(letterKey('n'))
	if !r.WaitFor("New service", 3*time.Second) {
		t.Fatalf("add-service modal did not open. Output:\n%s\n=== DECODED ===\n%s", r.Output(), r.scr.String())
	}

	// Name field is focused first: type the service name, Tab to the image
	// field, type the reference, Enter submits.
	for _, ch := range "proxy" {
		r.Send(letterKey(ch))
	}
	r.Send(keyPress(tea.KeyTab))
	for _, ch := range "traefik:v3" {
		r.Send(letterKey(ch))
	}
	r.Send(keyPress(tea.KeyEnter))

	if !r.WaitForNot("New service", 3*time.Second) {
		t.Fatalf("add-service modal did not close. Output:\n%s", r.Output())
	}

	// The inline editor opens on the new service with its minimal fragment -
	// this is the assertion that the selection/focus/edit-request race
	// resolved correctly.
	if !r.WaitFor("image: traefik:v3", 3*time.Second) {
		t.Fatalf("inline editor did not open on the new service. Output:\n%s", r.Output())
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		contents, err := os.ReadFile(filepath.Join(dir, "compose.yaml"))
		if err == nil && strings.Contains(string(contents), "proxy:") && strings.Contains(string(contents), "traefik:v3") {
			return // success
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("compose.yaml was not written with the new service to %s", dir)
}

// setupProjectDir drops a minimal compose project in a temp dir and moves
// the test there. Extracted here so other rig panel-key tests can reuse it.
func setupProjectDir(t *testing.T) {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(panelKeyFixture), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	t.Chdir(dir)
}

const panelKeyFixture = `services:
  web:
    image: nginx:alpine
    profiles: ["core"]
`
