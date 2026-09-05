package apptypes

// The one answer to "is this service running", and the one way to find a
// service's container.
//
// There used to be four answers. Two sites reported running when *any*
// container for the service was running; two returned on the *first* container
// found, whatever its state. One container per service makes those the same
// answer, which is why it went unnoticed - but a stopped container left beside
// a fresh one after a restart is enough to separate them, and then the same
// screen contradicted itself: the group table called a service stopped while
// the details panel beside it called the same service running.
//
// The rule is *any running container means running*. It is the reading that
// survives a leftover container, and it does not pretend cais manages
// replicas - only that it does not lie about them.

// ServiceRunning reports whether serviceName has at least one running
// container.
func ServiceRunning(containers []DockerContainer, serviceName string) bool {
	for _, c := range containers {
		if c.Service == serviceName && c.State == "running" {
			return true
		}
	}

	return false
}

// ContainerForService returns a container belonging to serviceName.
//
// It prefers a running one, so a caller rendering a single row's image, uptime
// or ports describes the live container rather than a stale one that happens to
// come first in docker's output. Falls back to the first container found, since
// a stopped service still has details worth showing.
func ContainerForService(containers []DockerContainer, serviceName string) (DockerContainer, bool) {
	var first DockerContainer
	var found bool

	for _, c := range containers {
		if c.Service != serviceName {
			continue
		}
		if c.State == "running" {
			return c, true
		}
		if !found {
			first, found = c, true
		}
	}

	return first, found
}
