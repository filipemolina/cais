package utils

import (
	"strings"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
)

func TestTemplatesForOrdersImageMatchesBeforeGeneric(t *testing.T) {
	names := func(ts []HealthcheckTemplate) []string {
		var out []string
		for _, t := range ts {
			out = append(out, t.Name)
		}
		return out
	}

	cases := []struct {
		image string
		want  []string
	}{
		{"postgres:16", []string{"PostgreSQL", "Generic HTTP"}},
		{"library/postgres:16", []string{"PostgreSQL", "Generic HTTP"}},
		{"lscr.io/linuxserver/mariadb:latest", []string{"MariaDB", "Generic HTTP"}},
		{"redis:7-alpine", []string{"Redis", "Generic HTTP"}},
		{"nginx:alpine", []string{"nginx", "Generic HTTP"}},
		// An unmatched image gets the generic fallback and nothing else -
		// never filtered out entirely.
		{"ghcr.io/someone/unknown-app:latest", []string{"Generic HTTP"}},
	}

	for _, tc := range cases {
		t.Run(tc.image, func(t *testing.T) {
			got := names(TemplatesFor(tc.image))
			if len(got) != len(tc.want) {
				t.Fatalf("TemplatesFor(%q) = %v, want %v", tc.image, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("TemplatesFor(%q)[%d] = %q, want %q", tc.image, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestDefaultGenericPortPrefillsFromFirstPublishedTarget(t *testing.T) {
	svc := types.ServiceConfig{
		Ports: []types.ServicePortConfig{
			{Published: "18096", Target: 8096},
			{Published: "18920", Target: 8920},
		},
	}

	if got := DefaultGenericPort(svc); got != "8096" {
		t.Errorf("DefaultGenericPort = %q, want 8096 (the first port's target)", got)
	}
}

func TestDefaultGenericPortFallsBackTo80(t *testing.T) {
	if got := DefaultGenericPort(types.ServiceConfig{}); got != "80" {
		t.Errorf("DefaultGenericPort with no ports = %q, want 80", got)
	}
}

func postgresTemplate() HealthcheckTemplate {
	for _, t := range HealthcheckCatalog {
		if t.Name == "PostgreSQL" {
			return t
		}
	}
	panic("PostgreSQL template not found")
}

func genericTemplate() HealthcheckTemplate {
	for _, t := range HealthcheckCatalog {
		if t.Generic {
			return t
		}
	}
	panic("no generic template found")
}

func TestApplyHealthcheckInsertsANewCheck(t *testing.T) {
	path := writeFixture(t, "services:\n  db:\n    image: postgres:16\n")

	if err := ApplyHealthcheck(path, "db", postgresTemplate(), ""); err != nil {
		t.Fatalf("ApplyHealthcheck: %v", err)
	}

	after := readFile(t, path)
	for _, want := range []string{"healthcheck:", "pg_isready -h 127.0.0.1 -p 5432", "interval: 30s", "start_period: 10s"} {
		if !strings.Contains(after, want) {
			t.Errorf("missing %q, got:\n%s", want, after)
		}
	}
	if strings.Contains(after, "start_interval") {
		t.Errorf("start_interval must never be emitted (version skew, D-item 6), got:\n%s", after)
	}
}

// A service that already has a healthcheck: gets it replaced, not
// duplicated - two healthcheck: keys in one service is a YAML error.
func TestApplyHealthcheckReplacesAnExistingCheck(t *testing.T) {
	path := writeFixture(t, `services:
  db:
    image: postgres:16
    healthcheck:
      test: ["CMD", "true"]
      interval: 5s
`)

	if err := ApplyHealthcheck(path, "db", postgresTemplate(), ""); err != nil {
		t.Fatalf("ApplyHealthcheck: %v", err)
	}

	after := readFile(t, path)
	if strings.Count(after, "healthcheck:") != 1 {
		t.Fatalf("expected exactly one healthcheck: key, got:\n%s", after)
	}
	if !strings.Contains(after, "pg_isready") {
		t.Errorf("old check was not replaced, got:\n%s", after)
	}
	if strings.Contains(after, `"true"`) {
		t.Errorf("old check's test still present, got:\n%s", after)
	}
}

// The generic template substitutes the given port into its CMD-SHELL string.
func TestApplyHealthcheckGenericSubstitutesThePort(t *testing.T) {
	path := writeFixture(t, "services:\n  web:\n    image: myapp:latest\n")

	if err := ApplyHealthcheck(path, "web", genericTemplate(), "8080"); err != nil {
		t.Fatalf("ApplyHealthcheck: %v", err)
	}

	after := readFile(t, path)
	if !strings.Contains(after, "http://127.0.0.1:8080/") {
		t.Errorf("port was not substituted, got:\n%s", after)
	}
}

func TestApplyHealthcheckGenericRequiresAPort(t *testing.T) {
	path := writeFixture(t, "services:\n  web:\n    image: myapp:latest\n")
	before := readFile(t, path)

	if err := ApplyHealthcheck(path, "web", genericTemplate(), ""); err == nil {
		t.Fatal("expected an error inserting the generic template with no port")
	}
	if after := readFile(t, path); after != before {
		t.Errorf("a rejected insert changed the file:\n%s", after)
	}
}

// Other services, their comments, and their own key order are untouched.
func TestApplyHealthcheckLeavesOtherServicesUntouched(t *testing.T) {
	path := writeFixture(t, `services:
  db:
    image: postgres:16
  web:
    image: nginx:alpine # the frontend
    ports:
      - "8080:80"
`)

	if err := ApplyHealthcheck(path, "db", postgresTemplate(), ""); err != nil {
		t.Fatalf("ApplyHealthcheck: %v", err)
	}

	after := readFile(t, path)
	for _, untouched := range []string{"# the frontend", `"8080:80"`} {
		if !strings.Contains(after, untouched) {
			t.Errorf("inserting db's healthcheck lost %q, got:\n%s", untouched, after)
		}
	}
}

func TestApplyHealthcheckUnknownServiceIsRejected(t *testing.T) {
	path := writeFixture(t, "services:\n  db:\n    image: postgres:16\n")

	if err := ApplyHealthcheck(path, "nope", postgresTemplate(), ""); err == nil {
		t.Fatal("expected an error for an unknown service")
	}
}
