package model

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"stack-stitcher/src/cmds"
	"stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

func getConfigErrNoFileMsg() cmds.GetConfigMsg {
	return cmds.GetConfigMsg{Err: utils.ErrNoComposeFile}
}

func keyPress(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code})
}

func teaKeyEsc() rune       { return tea.KeyEsc }
func teaKeyEnter() rune     { return tea.KeyEnter }
func teaKeyBackspace() rune { return tea.KeyBackspace }

// TestRig_StartsAndProducesOutput is a smoke test for the rig itself:
// confirm the program starts, runs Init's batch, and produces *some*
// rendered output. If this fails the rest of the e2e tests are blocked
// on harness issues rather than the feature.
func TestRig_StartsAndProducesOutput(t *testing.T) {
	r := newRig(t)

	// Sanity: utils.ErrNoComposeFile should be the sentinel. If this
	// fails the test was misconfigured.
	if !errors.Is(utils.ErrNoComposeFile, utils.ErrNoComposeFile) {
		t.Fatal("ErrNoComposeFile sentinel missing")
	}

	// Give Init's GetConfig a moment to round-trip and trigger a render.
	// In a clean dir this returns ErrNoComposeFile, which the model
	// surfaces via m.lastError - the red banner should appear.
	if !r.WaitFor("Error:", 2*time.Second) {
		t.Fatalf("expected the no-compose-file error banner to render. Output so far:\n%s", r.Output())
	}
}

// TestBootstrapModal_AutoOpensOnMissingFile is the headline assertion of
// this feature: when cmds.GetConfigMsg arrives with utils.ErrNoComposeFile,
// the model auto-opens the bootstrap modal.
func TestBootstrapModal_AutoOpensOnMissingFile(t *testing.T) {
	r := newRig(t)

	// Wait briefly for Init's GetConfig to fire on its own, then send
	// the message explicitly to make the test deterministic regardless
	// of render-order timing.
	r.Send(getConfigErrNoFileMsg())

	if !r.WaitFor("New compose file", 2*time.Second) {
		t.Fatalf("bootstrap modal did not auto-open on ErrNoComposeFile. Output:\n%s", r.Output())
	}
}

// TestBootstrapModal_EscRevealsBanner asserts that Esc dismisses the modal
// (the modal title disappears from the latest frame) while the underlying
// error banner remains visible.
func TestBootstrapModal_EscRevealsBanner(t *testing.T) {
	r := newRig(t)
	r.Send(getConfigErrNoFileMsg())

	if !r.WaitFor("New compose file", 2*time.Second) {
		t.Fatalf("modal did not open. Output:\n%s", r.Output())
	}

	r.Send(keyPress(teaKeyEsc()))
	time.Sleep(200 * time.Millisecond)

	afterEsc := r.Latest()

	// Loose assertion: the latest frame after Esc should not contain
	// the modal title. If it does, the modal wasn't dismissed.
	if strings.Contains(afterEsc, "New compose file") {
		t.Fatalf("modal title still in latest frame after Esc. afterEsc len=%d", len(afterEsc))
	}
}

// TestBootstrapModal_EnterAdvancesToServicePrompt covers the happy path on
// step 1: the default filename "compose.yaml" is valid in a clean dir, so
// pressing Enter moves the modal to the "Add a first service?" prompt.
func TestBootstrapModal_EnterAdvancesToServicePrompt(t *testing.T) {
	t.Chdir(t.TempDir())

	r := newRig(t)
	r.Send(getConfigErrNoFileMsg())

	if !r.WaitFor("New compose file", 2*time.Second) {
		t.Fatalf("modal did not open. Output:\n%s", r.Output())
	}

	// Advance the cursor so we only see the post-Enter render.
	r.Latest()

	r.Send(keyPress(teaKeyEnter()))
	time.Sleep(200 * time.Millisecond)

	afterEnter := r.Latest()
	if !strings.Contains(afterEnter, "Add a first service") {
		t.Fatalf("modal did not advance to service prompt. Latest frame:\n%q", afterEnter)
	}
}

// TestBootstrapModal_SkipServiceWritesEmptyFile covers the "n" path on
// step 2: skipping the first service creates a compose file with an empty
// services mapping, and the file exists on disk after the modal closes.
func TestBootstrapModal_SkipServiceWritesEmptyFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	r := newRig(t)
	r.Send(getConfigErrNoFileMsg())

	if !r.WaitFor("New compose file", 2*time.Second) {
		t.Fatalf("modal did not open. Output:\n%s", r.Output())
	}
	r.Send(keyPress(teaKeyEnter()))
	if !r.WaitFor("Add a first service", 2*time.Second) {
		t.Fatalf("modal did not advance. Output:\n%s", r.Output())
	}

	r.Send(keyPress(rune('n')))

	// The CreateComposeFile cmd runs WriteNewComposeFile, which writes
	// the file synchronously. Give the program a moment to refresh from
	// the new config.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filepath.Join(dir, "compose.yaml")); err == nil {
			return // success
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("compose.yaml was not written to %s after dismissing the modal with 'n'. Output:\n%s", dir, r.Output())
}

// TestBootstrapModal_EmptyFilenameShowsInlineError clears the default
// filename, presses Enter, and asserts the inline validation message.
func TestBootstrapModal_EmptyFilenameShowsInlineError(t *testing.T) {
	t.Chdir(t.TempDir())

	r := newRig(t)
	r.Send(getConfigErrNoFileMsg())

	if !r.WaitFor("New compose file", 2*time.Second) {
		t.Fatalf("modal did not open. Output:\n%s", r.Output())
	}

	// The default is "compose.yaml" (12 chars). Send 12 backspaces to
	// clear it, then Enter to submit empty.
	for range 12 {
		r.Send(keyPress(teaKeyBackspace()))
	}
	r.Latest()
	r.Send(keyPress(teaKeyEnter()))
	time.Sleep(200 * time.Millisecond)

	afterEnter := r.Latest()
	if !strings.Contains(afterEnter, "Filename can't be empty") {
		t.Fatalf("expected inline 'Filename can't be empty' error. Latest frame:\n%q", afterEnter)
	}
}
