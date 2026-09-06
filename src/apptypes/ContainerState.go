package apptypes

import "strings"

// StateTier is what the UI actually distinguishes about a container's state.
//
// Docker reports seven states - created, restarting, running, removing,
// paused, exited and dead - and the app used to fold all of them into
// "running" or "stopped". That was survivable while the only question was
// "is it up", but it meant a container docker had given up on (dead) and one
// the user had deliberately stopped (exited) rendered identically, and a
// service mid-restart looked stopped rather than busy.
//
// The tiers exist so the colour choice is made once. What a tier *means* is
// the point; which colour draws it is chrome's business.
type StateTier int

const (
	// TierUnknown is "docker has not answered yet", not a state docker
	// reports. Rendering it as stopped is a guess dressed up as a fact -
	// see serviceslist.Model.containersKnown.
	TierUnknown StateTier = iota

	// TierRunning is the one state that means the service is up.
	TierRunning

	// TierTransitional is on its way somewhere under its own power:
	// restarting will likely be running shortly, paused is deliberately
	// frozen. Both want attention without claiming a fault.
	TierTransitional

	// TierInert exists but is not doing anything, and nobody needs to act:
	// created has never been started, removing is on its way out because
	// somebody asked for it.
	TierInert

	// TierStopped is the ordinary, expected stop - exited, or no container
	// at all. It is by far the most common non-running state, which is why
	// it does not get the fault colour.
	TierStopped

	// TierFault is dead: docker could not remove the container and cannot
	// act on it. This is the only tier that means something is wrong.
	TierFault
)

// ClassifyContainerState maps one of docker's container states to its tier.
//
// The match is case-insensitive: docker's JSON reports lowercase, but
// `docker inspect` and some API paths capitalise, and a state that fell
// through to the default would silently render as an ordinary stop.
//
// An unrecognised state is TierStopped rather than TierFault: a state docker
// adds later is far more likely to be benign than broken, and claiming a
// fault on a string we do not know is the worse failure.
func ClassifyContainerState(state string) StateTier {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "running":
		return TierRunning
	case "restarting", "paused":
		return TierTransitional
	case "created", "removing":
		return TierInert
	case "exited", "stopped":
		// "stopped" is not one of docker's words. It is the app's, for a
		// service with no container at all, and it means the same thing to
		// a reader as exited does.
		return TierStopped
	case "dead":
		return TierFault
	case "":
		return TierUnknown
	default:
		return TierStopped
	}
}

// ContainerStateFor returns the state of the container the UI should describe
// for serviceName, or "" when docker has not reported one.
//
// It reads through ContainerForService, so it describes the same container the
// row's image, uptime and ports came from rather than a different one.
func ContainerStateFor(containers []DockerContainer, serviceName string) string {
	c, ok := ContainerForService(containers, serviceName)
	if !ok {
		return ""
	}

	return c.State
}
