package utils

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

// DockerState is which of the five states a preflight probe found. See
// docs/plans/docker-preflight.md for the research behind the five.
type DockerState int

const (
	DockerOK DockerState = iota
	DockerNotInstalled
	DockerComposeMissing
	DockerDaemonUnreachable
	DockerPermissionDenied
)

// DockerStatus is the whole answer: the state, plus the facts worth
// reporting alongside it.
type DockerStatus struct {
	State          DockerState
	EngineVersion  string // "29.6.0", when known
	ComposeVersion string // "5.1.4",  when known
	Endpoint       string // DOCKER_HOST or the active context's endpoint
	ComposeV1Found bool   // docker-compose (hyphen) is on PATH
	Raw            string // the failing probe's output, for the modal's detail line
}

// probes is the seam that makes DockerPreflight testable: the real one
// shells out, classify's tests supply a struct literal.
type probes struct {
	lookPath       func(string) (string, error)
	composeV1Found func() bool
	composeVersion func() (string, error)
	engineVersion  func() (string, error)
	endpoint       func() string
}

// classify is pure: given probe results, which state is this? It has no
// os/exec import, so it is unit-tested without docker installed - see D1 in
// docs/plans/docker-preflight.md for why classification comes from which
// probe failed rather than from parsing what it printed.
//
// endpoint is only ever probed on the states that need it (3 and 4): a
// healthy machine must never pay for the extra `docker context inspect`.
func classify(p probes) DockerStatus {
	if _, err := p.lookPath("docker"); err != nil {
		status := DockerStatus{State: DockerNotInstalled}
		if isExecErrPermission(err) {
			status.Raw = "docker is on PATH but not executable"
		}
		return status
	}

	composeOut, composeErr := p.composeVersion()
	if composeErr != nil {
		return DockerStatus{
			State:          DockerComposeMissing,
			Raw:            composeOut,
			ComposeV1Found: p.composeV1Found(),
		}
	}

	engineOut, engineErr := p.engineVersion()
	if engineErr != nil {
		// Anything unrecognised falls through to DockerDaemonUnreachable,
		// whose remediation text ends with a pointer at socket permissions -
		// see D1. Docker rewords this sentence between majors (measured:
		// 28 -> 29 changed it entirely), so this must stay a substring check
		// on one documented marker, never a match on the whole message.
		state := DockerDaemonUnreachable
		if strings.Contains(engineOut, "permission denied") {
			state = DockerPermissionDenied
		}
		return DockerStatus{
			State:    state,
			Raw:      engineOut,
			Endpoint: p.endpoint(),
		}
	}

	return DockerStatus{
		State:          DockerOK,
		ComposeVersion: strings.TrimSpace(composeOut),
		EngineVersion:  strings.TrimSpace(engineOut),
	}
}

// DockerPreflight runs the real probes and classifies them.
func DockerPreflight() DockerStatus {
	return classify(probes{
		lookPath: exec.LookPath,
		composeV1Found: func() bool {
			_, err := exec.LookPath("docker-compose")
			return err == nil
		},
		composeVersion: func() (string, error) {
			out, err := exec.Command("docker", "compose", "version", "--short").CombinedOutput()
			return string(out), err
		},
		engineVersion: func() (string, error) {
			out, err := exec.Command("docker", "version", "--format", "{{.Server.Version}}").CombinedOutput()
			return string(out), err
		},
		endpoint: dockerEndpoint,
	})
}

// dockerEndpoint reports the daemon endpoint being dialled: DOCKER_HOST when
// set, otherwise the active context's endpoint. The env var is free; the
// context command costs a process, so classify only calls this at all once
// DOCKER_HOST has already come up empty and a probe has already failed.
func dockerEndpoint() string {
	if host := os.Getenv("DOCKER_HOST"); host != "" {
		return host
	}

	out, err := exec.Command("docker", "context", "inspect", "--format", "{{.Endpoints.docker.Host}}").CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// isExecErrPermission reports whether err is the "found but not executable"
// flavor of exec.LookPath's failure, as opposed to "not found at all". Docker
// present but with its execute bit stripped classifies as DockerNotInstalled
// too (there is no separate state for it - see the edge case in
// docs/plans/docker-preflight.md), but the detail is worth keeping for the
// modal rather than silently folding it into the generic message.
func isExecErrPermission(err error) bool {
	return errors.Is(err, os.ErrPermission)
}
