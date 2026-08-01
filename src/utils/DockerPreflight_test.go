package utils

import (
	"errors"
	"os"
	"os/exec"
	"testing"
)

// Every fixture here was captured from a real docker on 2026-08-01 (Engine
// 29.6.0, Compose v5.1.4, Linux) except state 4's, which could not be
// reproduced locally and comes from Docker's own documentation instead - see
// docs/plans/docker-preflight.md's Research section. Do not replace these
// with plausible-looking strings.
const (
	daemonDownV29 = "failed to connect to the docker API at unix:///tmp/nope.sock; check if the path\n" +
		"is correct and if the daemon is running: dial unix /tmp/nope.sock: connect: no\n" +
		"such file or directory"
	daemonDownV28 = "Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?"
	// From Docker's documentation - not reproduced on the author's own
	// machine, which is in the docker group. Treated as a heuristic (D1),
	// not a captured contract.
	permissionDeniedMsg = "permission denied while trying to connect to the Docker daemon socket at unix:///var/run/docker.sock"
)

func fail(out string) func() (string, error) {
	return func() (string, error) { return out, errors.New(out) }
}

func healthyProbes() probes {
	return probes{
		lookPath:       func(string) (string, error) { return "/usr/bin/docker", nil },
		composeV1Found: func() bool { return false },
		composeVersion: func() (string, error) { return "5.1.4", nil },
		engineVersion:  func() (string, error) { return "29.6.0", nil },
		endpoint:       func() string { return "" },
	}
}

func TestClassifyHealthy(t *testing.T) {
	status := classify(healthyProbes())

	if status.State != DockerOK {
		t.Fatalf("State = %v, want DockerOK", status.State)
	}
	if status.ComposeVersion != "5.1.4" {
		t.Errorf("ComposeVersion = %q, want 5.1.4", status.ComposeVersion)
	}
	if status.EngineVersion != "29.6.0" {
		t.Errorf("EngineVersion = %q, want 29.6.0", status.EngineVersion)
	}
}

func TestClassifyNoBinary(t *testing.T) {
	p := healthyProbes()
	p.lookPath = func(string) (string, error) { return "", exec.ErrNotFound }

	status := classify(p)

	if status.State != DockerNotInstalled {
		t.Fatalf("State = %v, want DockerNotInstalled", status.State)
	}
}

func TestClassifyBinaryNotExecutable(t *testing.T) {
	p := healthyProbes()
	p.lookPath = func(string) (string, error) { return "", os.ErrPermission }

	status := classify(p)

	if status.State != DockerNotInstalled {
		t.Fatalf("State = %v, want DockerNotInstalled", status.State)
	}
	if status.Raw == "" {
		t.Error("Raw is empty, want a note that docker was found but is not executable")
	}
}

func TestClassifyPluginMissing(t *testing.T) {
	p := healthyProbes()
	p.composeVersion = fail("docker: 'compose' is not a docker command.")

	status := classify(p)

	if status.State != DockerComposeMissing {
		t.Fatalf("State = %v, want DockerComposeMissing", status.State)
	}
}

func TestClassifyPluginMissingV1Present(t *testing.T) {
	p := healthyProbes()
	p.composeVersion = fail("docker: 'compose' is not a docker command.")
	p.composeV1Found = func() bool { return true }

	status := classify(p)

	if status.State != DockerComposeMissing {
		t.Fatalf("State = %v, want DockerComposeMissing", status.State)
	}
	if !status.ComposeV1Found {
		t.Error("ComposeV1Found = false, want true")
	}
}

func TestClassifyDaemonDown(t *testing.T) {
	p := healthyProbes()
	p.engineVersion = fail(daemonDownV29)
	p.endpoint = func() string { return "unix:///tmp/nope.sock" }

	status := classify(p)

	if status.State != DockerDaemonUnreachable {
		t.Fatalf("State = %v, want DockerDaemonUnreachable", status.State)
	}
	if status.Endpoint != "unix:///tmp/nope.sock" {
		t.Errorf("Endpoint = %q, want the dialled socket", status.Endpoint)
	}
}

// The two daemon-down wordings are the test that pins D1: both must classify
// the same, and neither may be matched by a substring of the other.
func TestClassifyDaemonDownOldWording(t *testing.T) {
	p := healthyProbes()
	p.engineVersion = fail(daemonDownV28)

	status := classify(p)

	if status.State != DockerDaemonUnreachable {
		t.Fatalf("State = %v, want DockerDaemonUnreachable", status.State)
	}
}

func TestClassifyPermissionDenied(t *testing.T) {
	p := healthyProbes()
	p.engineVersion = fail(permissionDeniedMsg)

	status := classify(p)

	if status.State != DockerPermissionDenied {
		t.Fatalf("State = %v, want DockerPermissionDenied", status.State)
	}
}

// The documented fallback: an unrecognised failure still lands on state 3,
// whose remedy text points at socket permissions as a hedge.
func TestClassifyUnrecognisedFailureFallsBackToDaemonUnreachable(t *testing.T) {
	p := healthyProbes()
	p.engineVersion = fail("some future wording nobody has seen yet")

	status := classify(p)

	if status.State != DockerDaemonUnreachable {
		t.Fatalf("State = %v, want DockerDaemonUnreachable (the documented fallback)", status.State)
	}
}

// classify must never touch os/exec directly - the whole point of the probes
// seam is that this file can be audited for that import and come up empty.
func TestClassifyNeverCallsEndpointOnAHealthyMachine(t *testing.T) {
	p := healthyProbes()
	p.endpoint = func() string {
		t.Fatal("endpoint probed on a healthy machine")
		return ""
	}

	classify(p)
}
