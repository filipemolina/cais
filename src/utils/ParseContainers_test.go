package utils

import "testing"

const arrayFixture = `[
  {"ID": "abc123", "Names": "stack-web-1", "Service": "web", "State": "running", "HealthStatus": "healthy"},
  {"ID": "def456", "Names": "stack-db-1", "Service": "db", "State": "exited"}
]`

const ndjsonFixture = `{"ID": "abc123", "Names": "stack-web-1", "Service": "web", "State": "running", "HealthStatus": "healthy"}
{"ID": "def456", "Names": "stack-db-1", "Service": "db", "State": "exited"}
`

func TestParseContainers_JSONArray(t *testing.T) {
	containers, err := ParseContainers(arrayFixture)
	if err != nil {
		t.Fatalf("ParseContainers: %v", err)
	}

	if len(containers) != 2 {
		t.Fatalf("got %d containers, want 2", len(containers))
	}

	if containers[0].ID != "abc123" || containers[0].Service != "web" || containers[0].State != "running" {
		t.Errorf("unexpected first container: %+v", containers[0])
	}

	if containers[1].ID != "def456" || containers[1].Service != "db" || containers[1].State != "exited" {
		t.Errorf("unexpected second container: %+v", containers[1])
	}
}

func TestParseContainers_NDJSON(t *testing.T) {
	containers, err := ParseContainers(ndjsonFixture)
	if err != nil {
		t.Fatalf("ParseContainers: %v", err)
	}

	if len(containers) != 2 {
		t.Fatalf("got %d containers, want 2", len(containers))
	}

	if containers[0].ID != "abc123" || containers[0].Service != "web" || containers[0].State != "running" {
		t.Errorf("unexpected first container: %+v", containers[0])
	}

	if containers[1].ID != "def456" || containers[1].Service != "db" || containers[1].State != "exited" {
		t.Errorf("unexpected second container: %+v", containers[1])
	}
}

func TestParseContainers_Empty(t *testing.T) {
	for _, output := range []string{"", "\n", "[]"} {
		containers, err := ParseContainers(output)
		if err != nil {
			t.Fatalf("ParseContainers(%q): %v", output, err)
		}

		if len(containers) != 0 {
			t.Errorf("ParseContainers(%q) = %v, want empty", output, containers)
		}
	}
}

func TestParseContainers_Invalid(t *testing.T) {
	if _, err := ParseContainers("not json"); err == nil {
		t.Errorf("expected an error for invalid output")
	}

	if _, err := ParseContainers("[{\"ID\": broken]"); err == nil {
		t.Errorf("expected an error for a malformed array")
	}
}
