package model

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components/helpoverlay"
	"github.com/filipemolina/stack-stitcher/src/utils"
)

// With the screen free, a foreground error takes the modal.
func TestForegroundErrorOpensModalWhenScreenIsFree(t *testing.T) {
	m := GetInitialModel(utils.ComposeSource{})

	cmd := m.reportForegroundError("docker start failed")

	if m.activeModal == nil {
		t.Fatal("expected an error modal")
	}
	if m.lastError != "" {
		t.Errorf("banner = %q, want the modal to carry the error alone", m.lastError)
	}
	if cmd != nil {
		t.Error("the modal path costs no banner row, so it needs no layout command")
	}
}

// The case the bare `activeModal == nil` guard dropped: press s, open the help
// overlay before the action comes back, and the failure arrives with the
// screen already taken. It used to vanish - no modal, no banner, a docker
// action that silently did nothing.
func TestForegroundErrorFallsBackToBannerWhenModalIsOpen(t *testing.T) {
	m := GetInitialModel(utils.ComposeSource{})
	m.activeModal = helpoverlay.New(m.helpContext(), m.config.configFiles, 80)
	openType := fmt.Sprintf("%T", m.activeModal)

	m.reportForegroundError("docker start failed")

	if got := fmt.Sprintf("%T", m.activeModal); got != openType {
		t.Errorf("modal = %s, want the %s the user opened deliberately", got, openType)
	}
	if m.lastError != "docker start failed" {
		t.Errorf("banner = %q, want the error to fall back to it", m.lastError)
	}
	if m.lastErrorFromPoll {
		t.Error("a foreground error must own the banner, or a later poll clears it")
	}
}

// The modal path leaves the banner alone, so it must leave the banner's owner
// alone too: a poll error still showing there is still the poll's to clear.
func TestForegroundErrorModalKeepsPollOwnershipOfTheBanner(t *testing.T) {
	m := GetInitialModel(utils.ComposeSource{})
	m.lastError = "docker daemon unavailable"
	m.lastErrorFromPoll = true

	m.reportForegroundError("docker start failed")

	if m.lastError != "docker daemon unavailable" {
		t.Errorf("banner = %q, want the poll error preserved", m.lastError)
	}
	if !m.lastErrorFromPoll {
		t.Error("the poll still owns the banner it put up")
	}
}

// A docker action failing while a modal is up reaches the user through the
// real Update path, not just the helper.
func TestDockerActionErrorReachesBannerThroughUpdate(t *testing.T) {
	m := GetInitialModel(utils.ComposeSource{})
	m.activeModal = helpoverlay.New(m.helpContext(), m.config.configFiles, 80)

	m = updateForTest(t, m, cmds.DockerActionMsg{Err: errors.New("docker start failed")})

	if !strings.Contains(m.lastError, "docker start failed") {
		t.Errorf("banner = %q, want the action error to have surfaced", m.lastError)
	}
}
