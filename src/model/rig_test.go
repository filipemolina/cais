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
	p := tea.NewProgram(
		GetInitialModel(utils.ComposeSource{}),
		tea.WithInput(nil),
		tea.WithOutput(out),
		tea.WithoutSignals(),
		tea.WithWindowSize(120, 40),
	)

	r := &rig{p: p, out: out, done: make(chan struct{})}
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

// Send injects a message into the program loop, the same way the
// tea.NewProgram API exposes for testing.
func (r *rig) Send(msg tea.Msg) {
	r.p.Send(msg)
}

// letterKey builds a KeyPressMsg for a printable character key, with both
// Code and Text set. key.Matches (used by the panel handlers) compares
// msg.String() against the binding strings, so a letter sent with only
// Code does not match - textinput is happy with Code alone, but the panels
// are not. Use this helper for any rig key that targets a panel binding
// (s/t/r/p/x/l/e/n/d); keep keyPress for special keys (esc, enter, tab,
// backspace) where Code is enough for textinput.
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

// WaitFor polls Latest() for substr until it appears or timeout elapses.
// Returns true if the substring was found.
func (r *rig) WaitFor(substr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(r.Latest(), substr) {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

// WaitForNot polls Latest() until substr is NOT in the latest chunk of
// output. Returns true if the substring disappeared within the timeout.
// Useful for asserting that a modal or banner has been dismissed.
func (r *rig) WaitForNot(substr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		latest := r.Latest()
		if !strings.Contains(latest, substr) {
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
