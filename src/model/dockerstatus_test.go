package model

import (
	"testing"

	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/aboutmodal"
	"github.com/filipemolina/cais/src/components/dockerstatusmodal"
	"github.com/filipemolina/cais/src/components/errormodal"
	"github.com/filipemolina/cais/src/utils"

	tea "charm.land/bubbletea/v2"
)

// A diagnosable DockerStatusMsg opens the diagnosis modal when nothing else
// owns the screen - the same path the startup probe and an error re-probe
// both take.
func TestDockerStatusMsgOpensTheDiagnosisModal(t *testing.T) {
	m := withGroupsLoaded(t)

	updated, cmd := m.Update(cmds.DockerStatusMsg{Status: utils.DockerStatus{State: utils.DockerDaemonUnreachable}})
	m = drive(updated, collect(cmd)...)

	if _, ok := m.activeModal.(dockerstatusmodal.Model); !ok {
		t.Fatalf("activeModal is %T, want dockerstatusmodal.Model", m.activeModal)
	}
}

// A healthy machine sees no new UI whatsoever (D3): the message that opens
// nothing is still the common case, on every startup.
func TestDockerStatusMsgHealthyOpensNothing(t *testing.T) {
	m := withGroupsLoaded(t)

	updated, cmd := m.Update(cmds.DockerStatusMsg{Status: utils.DockerStatus{State: utils.DockerOK}})
	m = drive(updated, collect(cmd)...)

	if m.activeModal != nil {
		t.Fatalf("activeModal = %T, want nil on a healthy probe", m.activeModal)
	}
}

// A background result has no business closing a modal the user is working
// in - the bootstrap-flakiness lesson (TODO.md), pinned here for the docker
// modal too: a failure arriving while an unrelated modal is open opens
// nothing and leaves that modal in place.
func TestDockerStatusMsgLeavesAnUnrelatedModalInPlace(t *testing.T) {
	m := withGroupsLoaded(t)

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	m = drive(updated, collect(cmd)...)
	if _, ok := m.activeModal.(aboutmodal.Model); !ok {
		t.Fatalf("precondition: activeModal is %T, want aboutmodal.Model", m.activeModal)
	}

	updated, cmd = m.Update(cmds.DockerStatusMsg{Status: utils.DockerStatus{State: utils.DockerDaemonUnreachable}})
	m = drive(updated, collect(cmd)...)

	if _, ok := m.activeModal.(aboutmodal.Model); !ok {
		t.Fatalf("activeModal is %T, want the About modal left untouched", m.activeModal)
	}
}

// The one modal a diagnosis IS allowed to replace: the raw-error modal that
// reportDockerError put up for the very same failure a moment before the
// re-probe resolved - see D4 in docs/plans/docker-preflight.md.
func TestDockerStatusMsgReplacesItsOwnRawErrorModal(t *testing.T) {
	m := withGroupsLoaded(t)
	m.activeModal = errormodal.New("docker compose ps failed: exit status 1", 120)

	updated, cmd := m.Update(cmds.DockerStatusMsg{Status: utils.DockerStatus{State: utils.DockerDaemonUnreachable}})
	m = drive(updated, collect(cmd)...)

	if _, ok := m.activeModal.(dockerstatusmodal.Model); !ok {
		t.Fatalf("activeModal is %T, want the raw-error modal replaced by dockerstatusmodal.Model", m.activeModal)
	}
}
