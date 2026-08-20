package utils

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/yaml.v3"
)

// HealthcheckTemplate is one row of the catalog: a probe the maintainers of
// a specific image ship, or (Generic) a best-effort HTTP probe for anything
// else. The catalog stays small and does not grow into a service directory
// (docs/plans/healthcheck-insertion.md, "The catalog stays at the 4+1 rows
// above") - every row is a correctness claim about an image, and a wrong
// one produces a container stuck at unhealthy forever.
type HealthcheckTemplate struct {
	Name        string   // shown in the picker
	Matches     []string // image substrings, e.g. {"postgres", "postgresql"}
	Test        []string // CMD/CMD-SHELL and its argument
	Interval    string
	Timeout     string
	Retries     uint64
	StartPeriod string
	// Generic asks the picker to collect the container-internal port before
	// building the test - the one field this catalog has, deliberately kept
	// to a single row rather than growing options per template.
	Generic bool
}

// HealthcheckCatalog is every template offered, in the order high-confidence
// image-specific probes are checked, ending with the generic fallback. Every
// probe here ships in the image it targets, needs no authentication, and
// omits start_interval - accepted by this app's own parser (compose-go
// v2.12.1) but not guaranteed to be by a user's docker compose CLI older
// than 2.20.2, so the app's own validation would pass a file the user's own
// tooling then rejects. Omitting it costs nothing.
var HealthcheckCatalog = []HealthcheckTemplate{
	{
		Name:        "PostgreSQL",
		Matches:     []string{"postgres", "postgresql"},
		Test:        []string{"CMD-SHELL", "pg_isready -h 127.0.0.1 -p 5432"},
		Interval:    "30s",
		Timeout:     "5s",
		Retries:     3,
		StartPeriod: "10s",
	},
	{
		Name:        "MariaDB",
		Matches:     []string{"mariadb"},
		Test:        []string{"CMD-SHELL", "healthcheck.sh --connect --innodb_initialized"},
		Interval:    "30s",
		Timeout:     "5s",
		Retries:     3,
		StartPeriod: "10s",
	},
	{
		Name:        "Redis",
		Matches:     []string{"redis"},
		Test:        []string{"CMD-SHELL", "redis-cli -h 127.0.0.1 ping | grep PONG"},
		Interval:    "30s",
		Timeout:     "5s",
		Retries:     3,
		StartPeriod: "10s",
	},
	{
		Name:        "nginx",
		Matches:     []string{"nginx"},
		Test:        []string{"CMD-SHELL", "wget -qO- http://127.0.0.1/ >/dev/null 2>&1"},
		Interval:    "30s",
		Timeout:     "5s",
		Retries:     3,
		StartPeriod: "10s",
	},
	{
		Name:        "Generic HTTP",
		Test:        []string{"CMD-SHELL", "wget -qO- http://127.0.0.1:%s/ >/dev/null 2>&1"},
		Interval:    "30s",
		Timeout:     "5s",
		Retries:     3,
		StartPeriod: "10s",
		Generic:     true,
	},
}

// TemplatesFor orders the catalog for a service: image-matched templates
// first (in catalog order), the generic fallback last. A service with no
// matching image-specific template still gets the generic one - it is
// never filtered out, only ever sorted to the end.
func TemplatesFor(image string) []HealthcheckTemplate {
	var matched, rest []HealthcheckTemplate

	for _, t := range HealthcheckCatalog {
		if t.Generic {
			rest = append(rest, t)
			continue
		}
		if imageMatches(image, t.Matches) {
			matched = append(matched, t)
		}
	}

	return append(matched, rest...)
}

func imageMatches(image string, substrings []string) bool {
	lower := strings.ToLower(image)
	for _, s := range substrings {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}

// DefaultGenericPort is the container-internal port the generic template's
// port field prefills with: the first ports: target if the service
// publishes one, else 80 - the common case for a bare web image.
func DefaultGenericPort(svc types.ServiceConfig) string {
	for _, p := range svc.Ports {
		if p.Target != 0 {
			return strconv.FormatUint(uint64(p.Target), 10)
		}
	}
	return "80"
}

// ApplyHealthcheck inserts or replaces the healthcheck: mapping under
// serviceName in the compose file at fileName. port is only used by
// templates with Generic set - it is substituted into Test's CMD-SHELL
// string - and is otherwise ignored.
//
// The healthcheck is built as a yaml.Node and inserted directly into the
// service's existing value node, the same read-modify-write shape
// utils.SetGroupMembers uses, rather than round-tripping through a whole-
// service fragment: a healthcheck is one mapping under the service, not the
// service itself.
func ApplyHealthcheck(fileName string, serviceName string, t HealthcheckTemplate, port string) error {
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
	if serviceValue.Kind != yaml.MappingNode {
		return fmt.Errorf("service %q is not a mapping", serviceName)
	}

	healthcheckNode, err := healthcheckNode(t, port)
	if err != nil {
		return err
	}

	setMappingValue(serviceValue, "healthcheck", healthcheckNode)

	candidate, err := encodeNode(doc)
	if err != nil {
		return err
	}

	if err := ValidateComposeCandidate(filepath.Dir(fileName), candidate); err != nil {
		return err
	}

	return ReplaceFileAtomically(fileName, candidate)
}

// healthcheckNode builds the healthcheck: mapping's value node from a
// template, field by field rather than through yaml.Marshal of a Go map -
// yaml.v3 sorts map keys alphabetically (interval, retries, start_period,
// test, timeout), and a human reading the file expects to see test first,
// the way every example in the compose spec writes it.
func healthcheckNode(t HealthcheckTemplate, port string) (*yaml.Node, error) {
	test := make([]string, len(t.Test))
	copy(test, t.Test)
	if t.Generic {
		if port == "" {
			return nil, fmt.Errorf("a port is required for %s", t.Name)
		}
		for i, s := range test {
			test[i] = strings.ReplaceAll(s, "%s", port)
		}
	}

	testSeq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, s := range test {
		testSeq.Content = append(testSeq.Content, scalarNode(s))
	}

	mapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	appendPair := func(key string, value *yaml.Node) {
		mapping.Content = append(mapping.Content, scalarNode(key), value)
	}
	appendPair("test", testSeq)
	appendPair("interval", scalarNode(t.Interval))
	appendPair("timeout", scalarNode(t.Timeout))
	appendPair("retries", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.FormatUint(t.Retries, 10)})
	appendPair("start_period", scalarNode(t.StartPeriod))

	return mapping, nil
}

func scalarNode(s string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s}
}

// setMappingValue sets key to value inside mapping, replacing the existing
// value node if key is already present (this is the "replace" the picker's
// hint refers to - a service may only carry one healthcheck: key) or
// appending a new key/value pair at the end otherwise.
func setMappingValue(mapping *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1] = value
			return
		}
	}

	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	mapping.Content = append(mapping.Content, keyNode, value)
}

// removeMappingValue deletes key from mapping if present. A no-op when the
// key is absent, so callers can call it unconditionally rather than
// checking first.
func removeMappingValue(mapping *yaml.Node, key string) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content = append(mapping.Content[:i], mapping.Content[i+2:]...)
			return
		}
	}
}
