package apptypes

import "testing"

// The case that separated the four implementations: a stopped container left
// beside a fresh running one, which is what a restart can leave behind. Two
// sites read this as running and two as stopped, so the group table called a
// service stopped while the details panel beside it called it running.
func TestServiceRunningIgnoresAStaleStoppedContainer(t *testing.T) {
	containers := []DockerContainer{
		{Service: "web", State: "exited"},
		{Service: "web", State: "running"},
	}

	if !ServiceRunning(containers, "web") {
		t.Error("a service with a running container is running, whatever else is lying around")
	}

	got, ok := ContainerForService(containers, "web")
	if !ok {
		t.Fatal("expected a container for web")
	}
	if got.State != "running" {
		t.Errorf("described the %q container, want the running one", got.State)
	}
}

// Order must not decide the answer, or the same screen disagrees with itself
// depending on what docker happened to print first.
func TestServiceRunningDoesNotDependOnContainerOrder(t *testing.T) {
	running := DockerContainer{Service: "web", State: "running"}
	stopped := DockerContainer{Service: "web", State: "exited"}

	if a, b := ServiceRunning([]DockerContainer{running, stopped}, "web"),
		ServiceRunning([]DockerContainer{stopped, running}, "web"); a != b {
		t.Errorf("order changed the answer: %v vs %v", a, b)
	}
}

func TestServiceWithNoRunningContainerIsNotRunning(t *testing.T) {
	containers := []DockerContainer{{Service: "web", State: "exited"}}

	if ServiceRunning(containers, "web") {
		t.Error("an exited container is not running")
	}

	// It is still the container to describe: a stopped service has an image
	// and a name worth showing.
	if got, ok := ContainerForService(containers, "web"); !ok || got.State != "exited" {
		t.Errorf("ContainerForService = (%+v, %v), want the exited container", got, ok)
	}
}

func TestAnUnknownServiceHasNoContainer(t *testing.T) {
	containers := []DockerContainer{{Service: "db", State: "running"}}

	if ServiceRunning(containers, "web") {
		t.Error("web has no container at all")
	}
	if _, ok := ContainerForService(containers, "web"); ok {
		t.Error("expected no container for web")
	}
}
