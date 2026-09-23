package detailspanel

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/compose-spec/compose-go/v2/types"

	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/utils"
)

const (
	localDigestFixture  = "sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6"
	remoteDigestFixture = "sha256:858f009f9709ce576febc734aa78b8f6d624b82571f9ddb6bda4377c833b3499"
)

// staleSelection is the panel with a service selected whose check came back
// Stale, carrying both digests.
func staleSelection() Model {
	service := types.ServiceConfig{Name: "web", Image: "nginx:alpine"}
	m := New(&service, "10.0.0.5").(Model)
	m = m.applySize().(Model)
	updated, _ := m.Update(cmds.SetImageUpdatesMsg{Updates: map[string]utils.ImageUpdate{
		"web": {
			Service:      "web",
			Image:        "nginx:alpine",
			Repo:         "library/nginx",
			LocalDigest:  localDigestFixture,
			RemoteDigest: remoteDigestFixture,
			State:        utils.ImageStale,
		},
	}})

	return updated.(Model)
}

// viewOf drives the panel into shape for one update map and strips the view.
func viewOf(updates map[string]utils.ImageUpdate) string {
	service := types.ServiceConfig{Name: "web", Image: "nginx:alpine"}
	m := New(&service, "").(Model)
	m = m.applySize().(Model)
	updated, _ := m.Update(cmds.SetImageUpdatesMsg{Updates: updates})

	return ansi.Strip(updated.(Model).View().Content)
}

// The header's status line names the update and carries the glyph after the
// state label; every other state adds nothing to the line. The Unknown case
// is the one that matters: a check that never answered must not read as an
// update, on the panel side any more than on the row side.
func TestTheHeaderNamesTheUpdateForAStaleService(t *testing.T) {
	stale := ansi.Strip(staleSelection().View().Content)
	if !strings.Contains(stale, "update available") {
		t.Errorf("a stale service's header does not name the update:\n%s", stale)
	}
	if !strings.Contains(stale, "↑") {
		t.Error("a stale service's header carries no glyph")
	}

	unknown := viewOf(map[string]utils.ImageUpdate{
		"web": {Service: "web", State: utils.ImageUnknown},
	})
	if strings.Contains(unknown, "update available") || strings.Contains(unknown, "↑") {
		t.Errorf("an Unknown answer read as an update:\n%s", unknown)
	}

	upToDate := viewOf(map[string]utils.ImageUpdate{
		"web": {Service: "web", LocalDigest: "sha256:a", RemoteDigest: "sha256:a", State: utils.ImageUpToDate},
	})
	if strings.Contains(upToDate, "update available") || strings.Contains(upToDate, "↑") {
		t.Errorf("an UpToDate answer read as an update:\n%s", upToDate)
	}
}

// The details panel is where the digests live un-shortened - the updates
// table truncates to twelve characters, this panel renders the value the
// column can hold (every long value in a prop table truncates to its
// column). Rows for a stale answer only: an equal pair is noise, Unknown
// has none.
func TestTheDigestRowsCarryTheUnshortenedValues(t *testing.T) {
	view := ansi.Strip(staleSelection().View().Content)
	if !strings.Contains(view, "Local digest") || !strings.Contains(view, "Remote digest") {
		t.Errorf("a stale service's details carry no digest rows:\n%s", view)
	}

	// Both digests visible well past the twelve characters the updates
	// table stops at, proving the panel does not pre-shorten.
	if !strings.Contains(view, "294b683cb724975bec92") {
		t.Errorf("the local digest was shortened: %s", view)
	}
	if !strings.Contains(view, "858f009f9709ce576feb") {
		t.Errorf("the remote digest was shortened: %s", view)
	}

	quiet := viewOf(map[string]utils.ImageUpdate{
		"web": {Service: "web", State: utils.ImageUnknown},
	})
	if strings.Contains(quiet, "Local digest") {
		t.Errorf("an Unknown answer grew digest rows:\n%s", quiet)
	}
}

// U on a stale service asks AppModel for the update; U on any other state is
// a no-op - the guard is what makes the key honest for the states the footer
// does not advertise it for.
func TestUpdateKeyAsksForTheUpdateOnlyForAStaleService(t *testing.T) {
	_, cmd := staleSelection().Update(keyPress('U'))
	if cmd == nil {
		t.Fatal("U on a stale service produced no command")
	}

	request, ok := findMessageOfType[cmds.RequestUpdateImageMsg](t, cmd)
	if !ok || request.Service != "web" {
		t.Fatalf("U on a stale service did not ask for the update, got %+v", request)
	}

	service := types.ServiceConfig{Name: "web", Image: "nginx:alpine"}
	m := New(&service, "").(Model)
	m = m.applySize().(Model)
	updated, _ := m.Update(cmds.SetImageUpdatesMsg{Updates: map[string]utils.ImageUpdate{
		"web": {Service: "web", State: utils.ImageUnknown},
	}})
	m = updated.(Model)

	_, cmd = m.Update(keyPress('U'))
	if cmd != nil {
		t.Error("U on an Unknown service produced a command")
	}
}
