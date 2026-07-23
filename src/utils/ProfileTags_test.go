package utils

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const baseFixture = `services:
  app:
    image: nginx:alpine
    profiles: ["core"] # core services
    ports:
      - "8085:80"

  db:
    image: postgres:alpine
    profiles: ["core"]

  cache:
    image: redis:alpine
`

const multiTagFixture = `services:
  app:
    image: nginx:alpine
    profiles: ["core", "extra"]

  db:
    image: postgres:alpine
    profiles: ["core"]
`

func writeFixture(t *testing.T, contents string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")

	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	return path
}

func readServiceProfiles(t *testing.T, path, service string) []string {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading result file: %v", err)
	}

	var doc struct {
		Services map[string]struct {
			Profiles []string `yaml:"profiles"`
		} `yaml:"services"`
	}

	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parsing result file: %v", err)
	}

	return doc.Services[service].Profiles
}

func hasProfilesKey(t *testing.T, path, service string) bool {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading result file: %v", err)
	}

	var doc map[string]map[string]map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parsing result file: %v", err)
	}

	_, ok := doc["services"][service]["profiles"]
	return ok
}

func TestAddProfileTag_NoExistingKey(t *testing.T) {
	path := writeFixture(t, baseFixture)

	if err := AddProfileTag(path, "extra", []string{"cache"}); err != nil {
		t.Fatalf("AddProfileTag: %v", err)
	}

	got := readServiceProfiles(t, path, "cache")
	want := []string{"extra"}

	if !slices.Equal(got, want) {
		t.Errorf("cache profiles = %v, want %v", got, want)
	}
}

func TestAddProfileTag_ExistingKeyPreservesComment(t *testing.T) {
	path := writeFixture(t, baseFixture)

	if err := AddProfileTag(path, "extra", []string{"app"}); err != nil {
		t.Fatalf("AddProfileTag: %v", err)
	}

	got := readServiceProfiles(t, path, "app")
	want := []string{"core", "extra"}

	if !slices.Equal(got, want) {
		t.Errorf("app profiles = %v, want %v", got, want)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading result file: %v", err)
	}

	if !strings.Contains(string(raw), "core services") {
		t.Errorf("expected line comment to survive the edit, got:\n%s", raw)
	}
}

func TestAddProfileTag_Idempotent(t *testing.T) {
	path := writeFixture(t, baseFixture)

	if err := AddProfileTag(path, "core", []string{"app"}); err != nil {
		t.Fatalf("AddProfileTag: %v", err)
	}

	got := readServiceProfiles(t, path, "app")
	want := []string{"core"}

	if !slices.Equal(got, want) {
		t.Errorf("app profiles = %v, want %v (should not duplicate)", got, want)
	}
}

func TestRemoveProfileTag_LastTagDropsKey(t *testing.T) {
	path := writeFixture(t, baseFixture)

	if err := RemoveProfileTag(path, "core"); err != nil {
		t.Fatalf("RemoveProfileTag: %v", err)
	}

	if hasProfilesKey(t, path, "app") {
		t.Errorf("expected app's profiles key to be removed entirely")
	}

	if hasProfilesKey(t, path, "db") {
		t.Errorf("expected db's profiles key to be removed entirely")
	}
}

func TestRemoveProfileTag_OneOfSeveral(t *testing.T) {
	path := writeFixture(t, multiTagFixture)

	if err := RemoveProfileTag(path, "extra"); err != nil {
		t.Fatalf("RemoveProfileTag: %v", err)
	}

	got := readServiceProfiles(t, path, "app")
	want := []string{"core"}

	if !slices.Equal(got, want) {
		t.Errorf("app profiles = %v, want %v", got, want)
	}

	if !hasProfilesKey(t, path, "app") {
		t.Errorf("expected app's profiles key to survive (still has %q)", "core")
	}

	dbGot := readServiceProfiles(t, path, "db")
	if !slices.Equal(dbGot, []string{"core"}) {
		t.Errorf("db profiles = %v, want unchanged [core]", dbGot)
	}
}
