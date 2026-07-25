package utils

import (
	"errors"
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

func readServiceGroups(t *testing.T, path, service string) []string {
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

func hasGroupsKey(t *testing.T, path, service string) bool {
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

func TestAddGroupTag_NoExistingKey(t *testing.T) {
	path := writeFixture(t, baseFixture)

	if err := AddGroupTag(path, "extra", []string{"cache"}); err != nil {
		t.Fatalf("AddGroupTag: %v", err)
	}

	got := readServiceGroups(t, path, "cache")
	want := []string{"extra"}

	if !slices.Equal(got, want) {
		t.Errorf("cache groups = %v, want %v", got, want)
	}
}

func TestAddGroupTag_ExistingKeyPreservesComment(t *testing.T) {
	path := writeFixture(t, baseFixture)

	if err := AddGroupTag(path, "extra", []string{"app"}); err != nil {
		t.Fatalf("AddGroupTag: %v", err)
	}

	got := readServiceGroups(t, path, "app")
	want := []string{"core", "extra"}

	if !slices.Equal(got, want) {
		t.Errorf("app groups = %v, want %v", got, want)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading result file: %v", err)
	}

	if !strings.Contains(string(raw), "core services") {
		t.Errorf("expected line comment to survive the edit, got:\n%s", raw)
	}
}

func TestAddGroupTag_Idempotent(t *testing.T) {
	path := writeFixture(t, baseFixture)

	if err := AddGroupTag(path, "core", []string{"app"}); err != nil {
		t.Fatalf("AddGroupTag: %v", err)
	}

	got := readServiceGroups(t, path, "app")
	want := []string{"core"}

	if !slices.Equal(got, want) {
		t.Errorf("app groups = %v, want %v (should not duplicate)", got, want)
	}
}

func TestRemoveGroupTag_LastTagDropsKey(t *testing.T) {
	path := writeFixture(t, baseFixture)

	if err := RemoveGroupTag(path, "core"); err != nil {
		t.Fatalf("RemoveGroupTag: %v", err)
	}

	if hasGroupsKey(t, path, "app") {
		t.Errorf("expected app's profiles key to be removed entirely")
	}

	if hasGroupsKey(t, path, "db") {
		t.Errorf("expected db's profiles key to be removed entirely")
	}
}

func TestRemoveGroupTag_OneOfSeveral(t *testing.T) {
	path := writeFixture(t, multiTagFixture)

	if err := RemoveGroupTag(path, "extra"); err != nil {
		t.Fatalf("RemoveGroupTag: %v", err)
	}

	got := readServiceGroups(t, path, "app")
	want := []string{"core"}

	if !slices.Equal(got, want) {
		t.Errorf("app groups = %v, want %v", got, want)
	}

	if !hasGroupsKey(t, path, "app") {
		t.Errorf("expected app's profiles key to survive (still has %q)", "core")
	}

	dbGot := readServiceGroups(t, path, "db")
	if !slices.Equal(dbGot, []string{"core"}) {
		t.Errorf("db groups = %v, want unchanged [core]", dbGot)
	}
}

func TestWriteNewComposeFile_Empty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")

	if err := WriteNewComposeFile(path, "", ""); err != nil {
		t.Fatalf("WriteNewComposeFile: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading result file: %v", err)
	}

	if !strings.Contains(string(raw), "services:") {
		t.Errorf("expected output to contain a top-level services: key, got:\n%s", raw)
	}
}

func TestWriteNewComposeFile_OneService(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")

	if err := WriteNewComposeFile(path, "app", "nginx:alpine"); err != nil {
		t.Fatalf("WriteNewComposeFile: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading result file: %v", err)
	}

	var doc struct {
		Services map[string]struct {
			Image string `yaml:"image"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parsing result file: %v", err)
	}

	got, ok := doc.Services["app"]
	if !ok {
		t.Fatalf("expected an app service in the output, got:\n%s", raw)
	}
	if got.Image != "nginx:alpine" {
		t.Errorf("app image = %q, want %q", got.Image, "nginx:alpine")
	}
}

func TestWriteNewComposeFile_ThenAddGroupTag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")

	if err := WriteNewComposeFile(path, "app", "nginx:alpine"); err != nil {
		t.Fatalf("WriteNewComposeFile: %v", err)
	}

	if err := AddGroupTag(path, "core", []string{"app"}); err != nil {
		t.Fatalf("AddGroupTag on freshly written file: %v", err)
	}

	got := readServiceGroups(t, path, "app")
	want := []string{"core"}
	if !slices.Equal(got, want) {
		t.Errorf("app groups = %v, want %v", got, want)
	}
}

func TestWriteNewComposeFile_RefusesExisting(t *testing.T) {
	path := writeFixture(t, baseFixture)

	err := WriteNewComposeFile(path, "app", "nginx:alpine")
	if err == nil {
		t.Fatalf("expected error when target file already exists")
	}
	if !errors.Is(err, os.ErrExist) {
		t.Errorf("expected wrapped os.ErrExist, got: %v", err)
	}
}
