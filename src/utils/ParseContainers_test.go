package utils

import "testing"

// The key is "Health", which is what `docker compose ps --format json`
// actually emits — the fixtures used to say "HealthStatus" (the Go field
// name), which no real Docker ever sends, so the tests passed while the
// HEALTH column was blank against a live daemon.
const arrayFixture = `[
  {"ID": "abc123", "Names": "stack-web-1", "Service": "web", "State": "running", "Health": "healthy"},
  {"ID": "def456", "Names": "stack-db-1", "Service": "db", "State": "exited"}
]`

const ndjsonFixture = `{"ID": "abc123", "Names": "stack-web-1", "Service": "web", "State": "running", "Health": "healthy"}
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

	if containers[0].HealthStatus != "healthy" {
		t.Errorf("HealthStatus = %q, want %q — the json tag is what binds compose's \"Health\" key", containers[0].HealthStatus, "healthy")
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

	if containers[0].HealthStatus != "healthy" {
		t.Errorf("HealthStatus = %q, want %q — the json tag is what binds compose's \"Health\" key", containers[0].HealthStatus, "healthy")
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
