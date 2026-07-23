package utils

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AddProfileTag tags each of the given services with profileName in the
// compose file at fileName, preserving the file's existing formatting and
// comments as much as possible. It's idempotent: a service that already
// carries the tag is left unchanged.
func AddProfileTag(fileName string, profileName string, serviceNames []string) error {
	doc, err := readComposeNode(fileName)
	if err != nil {
		return err
	}

	servicesNode, err := servicesMappingNode(doc)
	if err != nil {
		return err
	}

	for _, serviceName := range serviceNames {
		serviceNode := findMappingValue(servicesNode, serviceName)
		if serviceNode == nil {
			return fmt.Errorf("service %q not found in compose file", serviceName)
		}

		profilesNode := findMappingValue(serviceNode, "profiles")
		if profilesNode == nil {
			profilesNode = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
			serviceNode.Content = append(serviceNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "profiles"},
				profilesNode,
			)
		}

		if !sequenceContains(profilesNode, profileName) {
			profilesNode.Content = append(profilesNode.Content, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: profileName,
			})
		}
	}

	return writeComposeNode(fileName, doc)
}

// RemoveProfileTag strips profileName from every service in the compose
// file at fileName that carries it. A service's profiles key is removed
// entirely, rather than left as an empty list, once its last tag is gone.
func RemoveProfileTag(fileName string, profileName string) error {
	doc, err := readComposeNode(fileName)
	if err != nil {
		return err
	}

	servicesNode, err := servicesMappingNode(doc)
	if err != nil {
		return err
	}

	// Mapping content is a flat, alternating slice: Content[0] is a key,
	// Content[1] is its value, and so on.
	for i := 0; i+1 < len(servicesNode.Content); i += 2 {
		removeProfileFromService(servicesNode.Content[i+1], profileName)
	}

	return writeComposeNode(fileName, doc)
}

func removeProfileFromService(serviceNode *yaml.Node, profileName string) {
	for i := 0; i+1 < len(serviceNode.Content); i += 2 {
		if serviceNode.Content[i].Value != "profiles" {
			continue
		}

		profilesNode := serviceNode.Content[i+1]
		remaining := profilesNode.Content[:0]
		for _, item := range profilesNode.Content {
			if item.Value != profileName {
				remaining = append(remaining, item)
			}
		}
		profilesNode.Content = remaining

		if len(profilesNode.Content) == 0 {
			serviceNode.Content = append(serviceNode.Content[:i], serviceNode.Content[i+2:]...)
		}

		return
	}
}

func readComposeNode(fileName string) (*yaml.Node, error) {
	raw, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed reading %s: %w", fileName, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("failed parsing %s: %w", fileName, err)
	}

	return &doc, nil
}

func writeComposeNode(fileName string, doc *yaml.Node) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)

	if err := enc.Encode(doc); err != nil {
		return fmt.Errorf("failed encoding %s: %w", fileName, err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("failed encoding %s: %w", fileName, err)
	}

	if err := os.WriteFile(fileName, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed writing %s: %w", fileName, err)
	}

	return nil
}

func servicesMappingNode(doc *yaml.Node) (*yaml.Node, error) {
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("compose file is empty")
	}

	servicesNode := findMappingValue(doc.Content[0], "services")
	if servicesNode == nil {
		return nil, fmt.Errorf("compose file has no top-level services key")
	}

	return servicesNode, nil
}

// findMappingValue returns the value node for key in mapping, or nil if
// the key isn't present.
func findMappingValue(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}

	return nil
}

func sequenceContains(sequence *yaml.Node, value string) bool {
	for _, item := range sequence.Content {
		if item.Value == value {
			return true
		}
	}

	return false
}
