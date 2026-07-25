package utils

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ErrServiceRenamed is returned when an edited fragment comes back under a
// different service name. Renaming is not supported: other services may
// point at the old name in depends_on:, and a rename that leaves those
// dangling is worse than a refusal.
var ErrServiceRenamed = errors.New("renaming a service is not supported")

// ExtractServiceFragment returns the YAML for one service, as a single-key
// mapping exactly as it appears in the compose file:
//
//	web:
//	  image: nginx:alpine
//	  ports:
//	    - "8085:80"
//
// The service name is kept as the top-level key for two reasons. It gives
// the user the context they would have in the real file, and it gives
// callers somewhere to put an explanatory header comment that cannot leak
// back in: comments above the key attach to the key node, and
// ApplyServiceFragment only ever takes the value.
func ExtractServiceFragment(fileName string, serviceName string) ([]byte, error) {
	doc, err := readComposeNode(fileName)
	if err != nil {
		return nil, err
	}

	servicesNode, err := servicesMappingNode(doc)
	if err != nil {
		return nil, err
	}

	keyNode, valueNode := findMappingPair(servicesNode, serviceName)
	if valueNode == nil {
		return nil, fmt.Errorf("service %q not found in compose file", serviceName)
	}

	fragment := &yaml.Node{
		Kind:    yaml.MappingNode,
		Tag:     "!!map",
		Content: []*yaml.Node{keyNode, valueNode},
	}

	return encodeNode(fragment)
}

// ApplyServiceFragment parses an edited fragment and writes it back over
// serviceName in the compose file.
//
// The file is left untouched unless the fragment parses, is shaped like a
// service, and the whole resulting document still loads as compose. That
// last check is what makes editing a fragment safer than editing the file
// by hand.
func ApplyServiceFragment(fileName string, serviceName string, fragment []byte) error {
	editedValue, err := parseServiceFragment(serviceName, fragment)
	if err != nil {
		return err
	}

	doc, err := readComposeNode(fileName)
	if err != nil {
		return err
	}

	servicesNode, err := servicesMappingNode(doc)
	if err != nil {
		return err
	}

	// Replace the value in place, keeping the original key node so any
	// comments attached to it survive the edit.
	replaced := false
	for i := 0; i+1 < len(servicesNode.Content); i += 2 {
		if servicesNode.Content[i].Value == serviceName {
			servicesNode.Content[i+1] = editedValue
			replaced = true
			break
		}
	}

	if !replaced {
		return fmt.Errorf("service %q not found in compose file", serviceName)
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

// parseServiceFragment validates the shape of an edited fragment and
// returns the service's value node.
func parseServiceFragment(serviceName string, fragment []byte) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(fragment, &doc); err != nil {
		return nil, fmt.Errorf("edited service is not valid YAML: %w", err)
	}

	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("edited service %q is empty", serviceName)
	}

	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("edited service %q must be a %s: block, not a bare value", serviceName, serviceName)
	}

	// Content alternates key, value, so a single entry is two elements.
	if len(root.Content) != 2 {
		return nil, fmt.Errorf("edited service must contain exactly one service, found %d", len(root.Content)/2)
	}

	if name := root.Content[0].Value; name != serviceName {
		return nil, fmt.Errorf("%w: %q became %q", ErrServiceRenamed, serviceName, name)
	}

	value := root.Content[1]
	if value.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("edited service %q must be a block of keys, e.g. image:", serviceName)
	}

	return value, nil
}

// ValidateComposeCandidate reports whether contents would load as a compose
// file, without touching whatever is currently on disk.
//
// The candidate is written into dir rather than a system temp directory
// because compose resolves relative paths - build contexts, env_file: - from
// the compose file's own location, so validating anywhere else would reject
// files that are perfectly fine, and accept ones that aren't.
func ValidateComposeCandidate(dir string, contents []byte) error {
	temp, err := os.CreateTemp(dir, ".stack-stitcher-candidate-*.yaml")
	if err != nil {
		return fmt.Errorf("failed creating a file to validate against: %w", err)
	}
	tempName := temp.Name()

	defer func() {
		temp.Close()
		os.Remove(tempName)
	}()

	if _, err := temp.Write(contents); err != nil {
		return fmt.Errorf("failed writing the file to validate: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("failed writing the file to validate: %w", err)
	}

	if _, err := ReadConfigFile(tempName); err != nil {
		return err
	}

	return nil
}

// findMappingPair returns both the key and value nodes for key in mapping,
// or nils when it isn't present. The key node carries its own comments,
// which is why callers sometimes need it and not just the value.
func findMappingPair(mapping *yaml.Node, key string) (*yaml.Node, *yaml.Node) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i], mapping.Content[i+1]
		}
	}

	return nil, nil
}

// encodeNode renders a node at the indentation the rest of the file uses.
func encodeNode(node *yaml.Node) ([]byte, error) {
	var buf bytes.Buffer

	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)

	if err := enc.Encode(node); err != nil {
		return nil, fmt.Errorf("failed encoding YAML: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("failed encoding YAML: %w", err)
	}

	return buf.Bytes(), nil
}
