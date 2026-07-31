package utils

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AddGroupTag tags each of the given services with groupName in the
// compose file at fileName, preserving the file's existing formatting and
// comments as much as possible. It's idempotent: a service that already
// carries the tag is left unchanged.
func AddGroupTag(fileName string, groupName string, serviceNames []string) error {
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

		ensureGroupTag(serviceNode, groupName)
	}

	return writeComposeNode(fileName, doc)
}

// RemoveGroupTag strips groupName from every service in the compose
// file at fileName that carries it. A service's profiles key is removed
// entirely, rather than left as an empty list, once its last tag is gone.
func RemoveGroupTag(fileName string, groupName string) error {
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
		removeGroupFromService(servicesNode.Content[i+1], groupName)
	}

	return writeComposeNode(fileName, doc)
}

func removeGroupFromService(serviceNode *yaml.Node, groupName string) {
	for i := 0; i+1 < len(serviceNode.Content); i += 2 {
		if serviceNode.Content[i].Value != "profiles" {
			continue
		}

		profilesNode := serviceNode.Content[i+1]
		remaining := profilesNode.Content[:0]
		for _, item := range profilesNode.Content {
			if item.Value != groupName {
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

// writeComposeNode encodes doc into fileName, replacing it atomically. The
// document is encoded into memory first, so an encoding failure never
// reaches the user's compose file at all.
func writeComposeNode(fileName string, doc *yaml.Node) error {
	contents, err := encodeNode(doc)
	if err != nil {
		return fmt.Errorf("failed encoding %s: %w", fileName, err)
	}

	return ReplaceFileAtomically(fileName, contents)
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

// SetGroupMembers reconciles the compose file so that exactly the named
// services carry groupName as a profile tag. Services in members that lack
// the tag get it added; services not in members that carry it get it removed.
// A service whose profiles key becomes empty is cleaned up the same way
// RemoveGroupTag does.
//
// The reconciliation runs in a single read-modify-write pass rather than
// composing AddGroupTag and RemoveGroupTag, which would each open and close
// the file separately and leave a crash window with a half-applied edit.
func SetGroupMembers(fileName string, groupName string, members []string) error {
	doc, err := readComposeNode(fileName)
	if err != nil {
		return err
	}

	servicesNode, err := servicesMappingNode(doc)
	if err != nil {
		return err
	}

	memberSet := make(map[string]bool, len(members))
	for _, name := range members {
		memberSet[name] = true
	}

	for i := 0; i+1 < len(servicesNode.Content); i += 2 {
		serviceName := servicesNode.Content[i].Value
		serviceNode := servicesNode.Content[i+1]

		if memberSet[serviceName] {
			ensureGroupTag(serviceNode, groupName)
		} else {
			removeGroupFromService(serviceNode, groupName)
		}
	}

	return writeComposeNode(fileName, doc)
}

// RenameGroupTag replaces every profiles entry equal to oldName with
// newName in the compose file at fileName. Other profiles and every other
// key are untouched. Nothing else in a compose file references a profile
// by name (unlike service names, which depends_on references - which is
// why service renames are refused), so a rename cannot leave dangling
// references. Returns an error when no service carries oldName, which
// catches the file having changed since the caller last synced it.
func RenameGroupTag(fileName string, oldName string, newName string) error {
	doc, err := readComposeNode(fileName)
	if err != nil {
		return err
	}

	servicesNode, err := servicesMappingNode(doc)
	if err != nil {
		return err
	}

	renamed := false
	for i := 0; i+1 < len(servicesNode.Content); i += 2 {
		serviceNode := servicesNode.Content[i+1]
		profilesNode := findMappingValue(serviceNode, "profiles")
		if profilesNode == nil {
			continue
		}

		for _, item := range profilesNode.Content {
			if item.Value == oldName {
				item.Value = newName
				renamed = true
			}
		}
	}

	if !renamed {
		return fmt.Errorf("group %q not found in compose file", oldName)
	}

	return writeComposeNode(fileName, doc)
}

// ensureGroupTag adds groupName to serviceNode's profiles sequence if it is
// not already present. Creates the profiles key and sequence if absent.
func ensureGroupTag(serviceNode *yaml.Node, groupName string) {
	profilesNode := findMappingValue(serviceNode, "profiles")
	if profilesNode == nil {
		profilesNode = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		serviceNode.Content = append(serviceNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "profiles"},
			profilesNode,
		)
	}

	if !sequenceContains(profilesNode, groupName) {
		profilesNode.Content = append(profilesNode.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: groupName,
		})
	}
}

// WriteNewComposeFile creates a brand-new compose file at fileName with a
// top-level services mapping, optionally pre-seeded with one service. It
// refuses to overwrite an existing file: the caller is expected to have
// already shown a validation error in the modal in that case, so we surface
// os.ErrExist to make the failure mode explicit.
func WriteNewComposeFile(fileName string, serviceName string, image string) error {
	if _, err := os.Stat(fileName); err == nil {
		return fmt.Errorf("refusing to overwrite existing %s: %w", fileName, os.ErrExist)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("checking %s: %w", fileName, err)
	}

	doc := &yaml.Node{Kind: yaml.MappingNode}

	servicesValue := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if serviceName != "" {
		serviceMapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		serviceMapping.Content = append(serviceMapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: serviceName},
			&yaml.Node{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "image"},
					{Kind: yaml.ScalarNode, Value: image, Tag: "!!str"},
				},
			},
		)
		servicesValue.Content = append(servicesValue.Content, serviceMapping.Content...)
	}

	doc.Content = []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "services"},
		servicesValue,
	}

	return writeComposeNode(fileName, doc)
}
