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

// AddServiceFragment inserts a new service into the compose file at
// fileName.
//
// It is ApplyServiceFragment's opposite number: same fragment shape, same
// validation, same atomic write, but it refuses when the name is already
// taken instead of when it is absent. Insertion is at the end of the
// services: mapping, which is where a reader expects a new entry and the
// only position that never reorders the user's file.
func AddServiceFragment(fileName string, serviceName string, fragment []byte) error {
	newValue, err := parseServiceFragment(serviceName, fragment)
	if err != nil {
		return err
	}

	doc, err := readComposeNode(fileName)
	if err != nil {
		return err
	}

	servicesNode, err := addServicesMappingNode(doc)
	if err != nil {
		return err
	}

	for i := 0; i+1 < len(servicesNode.Content); i += 2 {
		if servicesNode.Content[i].Value == serviceName {
			return fmt.Errorf("service %q already exists in compose file", serviceName)
		}
	}

	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: serviceName}
	servicesNode.Content = append(servicesNode.Content, keyNode, newValue)

	candidate, err := encodeNode(doc)
	if err != nil {
		return err
	}

	if err := ValidateComposeCandidate(filepath.Dir(fileName), candidate); err != nil {
		return err
	}

	return ReplaceFileAtomically(fileName, candidate)
}

// DeleteService removes serviceName's entry from the services: mapping in
// fileName - AddServiceFragment's opposite number, minus a fragment to parse.
//
// The removal goes through the same validate-by-reload every other writer in
// this file uses, plus an explicit dependents check ValidateComposeCandidate
// alone cannot cover - see ensureNoDependents.
func DeleteService(fileName string, serviceName string) error {
	if err := ensureNoDependents(fileName, serviceName); err != nil {
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

	removed := false
	for i := 0; i+1 < len(servicesNode.Content); i += 2 {
		if servicesNode.Content[i].Value == serviceName {
			servicesNode.Content = append(servicesNode.Content[:i], servicesNode.Content[i+2:]...)
			removed = true
			break
		}
	}

	if !removed {
		return fmt.Errorf("service %q not found in compose file", serviceName)
	}

	candidate, err := encodeNode(doc)
	if err != nil {
		return err
	}

	// Still run the general reload check too: it is what catches every other
	// way a removal could leave the document broken (e.g. a network: or
	// volume: block left orphaned), the same guarantee every other writer in
	// this file gives.
	if err := ValidateComposeCandidate(filepath.Dir(fileName), candidate); err != nil {
		return err
	}

	return ReplaceFileAtomically(fileName, candidate)
}

// ensureNoDependents refuses to delete a service another one still names in
// depends_on:.
//
// This cannot be left to ValidateComposeCandidate's reload-and-check alone.
// compose-go's own consistency check (loader.checkConsistency) only walks
// project.Services - the profile-enabled set for whatever profile (if any)
// the load was asked for - and ReadConfigFile asks for none, so a service
// carrying a profiles: tag is loaded into project.DisabledServices instead
// and is never visited by that check. depends_on between two members of the
// same group is exactly this shape (both tagged, neither in project.Services
// without --profile), which is the common case for cais, not an edge case -
// so relying on the reload alone would silently accept a deletion that
// leaves a grouped service's depends_on dangling. This checks every service
// regardless of profile, merging Services and DisabledServices the same way
// AppModel.configSyncCmds does to build the full list cais shows.
func ensureNoDependents(fileName, serviceName string) error {
	project, err := ReadConfigFile(fileName)
	if err != nil {
		return err
	}

	for name, svc := range project.Services {
		if name == serviceName {
			continue
		}
		if _, ok := svc.DependsOn[serviceName]; ok {
			return fmt.Errorf("service %q depends on %q: remove that depends_on entry (or delete %q too) before deleting %q", name, serviceName, name, serviceName)
		}
	}
	for name, svc := range project.DisabledServices {
		if name == serviceName {
			continue
		}
		if _, ok := svc.DependsOn[serviceName]; ok {
			return fmt.Errorf("service %q depends on %q: remove that depends_on entry (or delete %q too) before deleting %q", name, serviceName, name, serviceName)
		}
	}

	return nil
}

// addServicesMappingNode is servicesMappingNode's insertion counterpart: it
// returns doc's services: mapping node, creating it (as an empty mapping)
// when the key is absent, and replacing it in place when present but null -
// a compose file with only name:/volumes: and no services: yet is legal,
// and AddServiceFragment is the one writer in this package that has to
// succeed on it. Every other writer only ever edits an existing service, so
// they keep using the stricter servicesMappingNode.
func addServicesMappingNode(doc *yaml.Node) (*yaml.Node, error) {
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("compose file is empty")
	}
	root := doc.Content[0]

	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value != "services" {
			continue
		}

		value := root.Content[i+1]
		if value.Kind != yaml.MappingNode {
			// services: present but null (or some other bare scalar) -
			// replace it with an empty mapping in place, keeping the key
			// node (and any comment attached to it) untouched.
			*value = yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		}
		return value, nil
	}

	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "services"}
	valueNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	root.Content = append(root.Content, keyNode, valueNode)
	return valueNode, nil
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
	temp, err := os.CreateTemp(dir, ".cais-candidate-*.yaml")
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
