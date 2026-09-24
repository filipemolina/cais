package cmds

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"

	"github.com/filipemolina/cais/src/utils"
)

// runCheck runs the check command and returns its answer, or an empty answer
// when the command built nothing (no project loaded).
func runCheck(t *testing.T, services []types.ServiceConfig, loaded bool) ImageUpdatesMsg {
	t.Helper()

	cmd := CheckImageUpdates(services, loaded)
	if cmd == nil {
		return ImageUpdatesMsg{}
	}

	msg := cmd()
	if msg == nil {
		t.Fatal("the check produced no message")
	}

	answer, ok := msg.(ImageUpdatesMsg)
	if !ok {
		t.Fatalf("the check produced %T, want ImageUpdatesMsg", msg)
	}

	return answer
}

// A project that is not loaded checks nothing: there is no file to key an
// answer to.
func TestCheckImageUpdatesNeedsAProject(t *testing.T) {
	if cmd := CheckImageUpdates(nil, false); cmd != nil {
		t.Fatal("an unloaded project dispatched a check")
	}
}

// A pure-digest reference files as Unknown without spending the network call:
// a digest is immutable and there is nothing to compare (D4). A service with
// no image: is skipped outright, not Unknown - there is no reference to
// check.
//
// Every service in these fixtures carries a pure digest or nothing, so the
// test never reaches the network: the pinned half skips it, and the no-image
// half has nothing to spend a call on.
func TestCheckImageUpdatesFilesPinnedAndSkipsImageless(t *testing.T) {
	none := types.ServiceConfig{Name: "built", Build: &types.BuildConfig{Context: "./app"}}
	pinned := types.ServiceConfig{
		Name:  "pinned",
		Image: "redis@sha256:6ab0b6e7381779332f97b8ca76193e45b0756f38d4c0dcda72dbb3c32061ab99",
	}

	answer := runCheck(t, []types.ServiceConfig{none, pinned}, true)

	if _, ok := answer.Updates["built"]; ok {
		t.Error("a service with no image: is listed somewhere; it is not applicable")
	}
	got, ok := answer.Updates["pinned"]
	if !ok {
		t.Fatal("a pure-digest reference vanished from the answer")
	}
	if got.State != utils.ImageUnknown {
		t.Errorf("a pure-digest reference read %v, want Unknown", got.State)
	}
	if got.LocalDigest != "" || got.RemoteDigest != "" {
		t.Errorf("a pure-digest reference spent the halves: %+v", got)
	}
}
