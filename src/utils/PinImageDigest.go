package utils

import (
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// PinImageDigest rewrites one service's image: value to the same reference
// with a digest appended — `redis:7-alpine` becomes
// `redis:7-alpine@sha256:…` — and writes the result through the same
// read-modify-write → validate → atomic-replace path every compose writer in
// this package uses. The snapshot ReplaceFileAtomically takes is the whole
// rollback story: no special backup code exists or needs to.
//
// The digest is the full "sha256:…" form the check's ImageUpdate carries.
// checkDigest refusing anything else is what keeps a caller from appending a
// bare hex string and leaving `image@858f00…` — a line compose would reject
// — in the user's file. A service that is not in the file, or has no image:
// key to pin, is refused for the same reason.
func PinImageDigest(fileName string, serviceName string, digest string) error {
	if err := checkDigest(digest); err != nil {
		return fmt.Errorf("cannot pin %s: %w", serviceName, err)
	}

	doc, err := readComposeNode(fileName)
	if err != nil {
		return err
	}

	servicesNode, err := servicesMappingNode(doc)
	if err != nil {
		return err
	}

	_, valueNode := findMappingPair(servicesNode, serviceName)
	if valueNode == nil {
		return fmt.Errorf("service %q not found in compose file", serviceName)
	}

	_, imageNode := findMappingPair(valueNode, "image")
	if imageNode == nil {
		return fmt.Errorf("service %q has no image: key to pin", serviceName)
	}
	if imageNode.Kind != yaml.ScalarNode {
		return fmt.Errorf("service %q's image: is not a plain value", serviceName)
	}

	// The written form is preserved as-is, host and tag included, with any
	// existing digest swapped: a re-pin replaces the old digest rather than
	// stacking a second @ clause onto it. Cutting at the first @ keeps the
	// user's reference exactly as they wrote it.
	name, _, _ := strings.Cut(imageNode.Value, "@")
	if name == "" {
		return fmt.Errorf("service %q's image: value %q is not a reference to pin", serviceName, imageNode.Value)
	}
	imageNode.Value = name + "@" + digest
	imageNode.Tag = "!!str"

	candidate, err := encodeNode(doc)
	if err != nil {
		return err
	}

	if err := ValidateComposeCandidate(filepath.Dir(fileName), candidate); err != nil {
		return err
	}

	return ReplaceFileAtomically(fileName, candidate)
}
