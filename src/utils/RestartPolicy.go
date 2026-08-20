package utils

import (
	"fmt"
	"path/filepath"
	"slices"
)

// restartPolicyCycle is the order B on the Services details panel steps
// through. "" stands for no restart: key at all - equivalent to the
// explicit "no" compose accepts, but omitting the key is the cleaner write
// when a service has never carried one.
var restartPolicyCycle = []string{"", "on-failure", "unless-stopped", "always"}

// NextRestartPolicy returns the policy after current in restartPolicyCycle,
// wrapping around. "no" is treated the same as "": both are the cycle's
// unset state, since a service written by hand with restart: no means
// exactly what an absent key means. A value outside the cycle (hand-edited,
// or a future compose keyword this app does not know) resets to the first
// entry rather than guessing where it belongs.
func NextRestartPolicy(current string) string {
	if current == "no" {
		current = ""
	}

	idx := slices.Index(restartPolicyCycle, current)
	if idx == -1 {
		return restartPolicyCycle[0]
	}

	return restartPolicyCycle[(idx+1)%len(restartPolicyCycle)]
}

// SetRestartPolicy writes policy into serviceName's restart: field in the
// compose file at fileName, through the same read-modify-write shape
// ApplyHealthcheck uses. An empty policy removes the key entirely rather
// than writing restart: "" - compose has no such value, and dropping the
// key is what "no policy" looks like in a file a human reads afterward.
func SetRestartPolicy(fileName string, serviceName string, policy string) error {
	doc, err := readComposeNode(fileName)
	if err != nil {
		return err
	}

	servicesNode, err := servicesMappingNode(doc)
	if err != nil {
		return err
	}

	_, serviceValue := findMappingPair(servicesNode, serviceName)
	if serviceValue == nil {
		return fmt.Errorf("service %q not found in compose file", serviceName)
	}

	if policy == "" {
		removeMappingValue(serviceValue, "restart")
	} else {
		setMappingValue(serviceValue, "restart", scalarNode(policy))
	}

	candidate, err := encodeNode(doc)
	if err != nil {
		return err
	}

	if err := ValidateComposeCandidate(filepath.Dir(fileName), candidate); err != nil {
		return err
	}

	return ReplaceFileAtomically(fileName, candidate)
}
