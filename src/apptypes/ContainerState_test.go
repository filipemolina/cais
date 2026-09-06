package apptypes

import "testing"

// Every state docker reports has to land somewhere deliberate. The app used to
// fold all seven into running-or-stopped, which is how a dead container and a
// deliberately stopped one came to look identical.
func TestEveryDockerStateIsClassified(t *testing.T) {
	cases := map[string]StateTier{
		"running":    TierRunning,
		"restarting": TierTransitional,
		"paused":     TierTransitional,
		"created":    TierInert,
		"removing":   TierInert,
		"exited":     TierStopped,
		"dead":       TierFault,

		// Not docker's word - the app's, for a service with no container.
		"stopped": TierStopped,

		// Docker has not answered yet. Distinct from every real state,
		// because "we do not know" is not "it is stopped".
		"": TierUnknown,
	}

	for state, want := range cases {
		if got := ClassifyContainerState(state); got != want {
			t.Errorf("ClassifyContainerState(%q) = %d, want %d", state, got, want)
		}
	}
}

// docker's JSON reports lowercase, but `docker inspect` and some API paths
// capitalise. A state that fell through on case would render as an ordinary
// stop, which is exactly the misreport this whole tier system exists to stop.
func TestClassifyIgnoresCaseAndSurroundingSpace(t *testing.T) {
	for _, state := range []string{"Running", "RUNNING", " running ", "\trunning"} {
		if got := ClassifyContainerState(state); got != TierRunning {
			t.Errorf("ClassifyContainerState(%q) = %d, want TierRunning", state, got)
		}
	}

	if got := ClassifyContainerState("Dead"); got != TierFault {
		t.Errorf("ClassifyContainerState(\"Dead\") = %d, want TierFault", got)
	}
}

// A state docker adds in some future version is far more likely to be benign
// than broken. Claiming a fault on a string we do not recognise would put a
// red cross on a healthy service.
func TestAnUnknownStateIsNotAFault(t *testing.T) {
	got := ClassifyContainerState("some-state-docker-invented-later")

	if got == TierFault {
		t.Error("an unrecognised state was reported as a fault")
	}
	if got != TierStopped {
		t.Errorf("got tier %d, want TierStopped", got)
	}
}

// ContainerStateFor has to describe the same container the row's image, uptime
// and ports came from, or the pill contradicts the table beside it.
func TestContainerStateForPrefersTheRunningContainer(t *testing.T) {
	containers := []DockerContainer{
		{Service: "web", State: "exited"},
		{Service: "web", State: "running"},
	}

	if got := ContainerStateFor(containers, "web"); got != "running" {
		t.Errorf("ContainerStateFor = %q, want %q", got, "running")
	}

	if got := ContainerStateFor(containers, "absent"); got != "" {
		t.Errorf("a service with no container reported %q, want the empty unknown", got)
	}
}
