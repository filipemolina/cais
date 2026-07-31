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

func TestSetGroupMembers_AddsAndRemovesInOnePass(t *testing.T) {
	path := writeFixture(t, baseFixture)

	// "core" originally tags app and db. Move it to db and cache only.
	if err := SetGroupMembers(path, "core", []string{"db", "cache"}); err != nil {
		t.Fatalf("SetGroupMembers: %v", err)
	}

	appGot := readServiceGroups(t, path, "app")
	if len(appGot) != 0 {
		t.Errorf("app should have lost the core tag, got %v", appGot)
	}
	if hasGroupsKey(t, path, "app") {
		t.Error("app's profiles key should be removed entirely once empty")
	}

	dbGot := readServiceGroups(t, path, "db")
	if !slices.Equal(dbGot, []string{"core"}) {
		t.Errorf("db groups = %v, want [core]", dbGot)
	}

	cacheGot := readServiceGroups(t, path, "cache")
	if !slices.Equal(cacheGot, []string{"core"}) {
		t.Errorf("cache groups = %v, want [core]", cacheGot)
	}
}

func TestSetGroupMembers_EmptyRemovesAll(t *testing.T) {
	path := writeFixture(t, baseFixture)

	// Removing every member of a group is the delete-group path expressed
	// through the same reconciliation, so it has to clean up the same way.
	if err := SetGroupMembers(path, "core", nil); err != nil {
		t.Fatalf("SetGroupMembers: %v", err)
	}

	if hasGroupsKey(t, path, "app") {
		t.Error("app's profiles key should be removed when the group is emptied")
	}
	if hasGroupsKey(t, path, "db") {
		t.Error("db's profiles key should be removed when the group is emptied")
	}
}

func TestSetGroupMembers_PreservesOtherTags(t *testing.T) {
	path := writeFixture(t, multiTagFixture)

	// Reconcile "core" to only db: app loses core but keeps "extra".
	if err := SetGroupMembers(path, "core", []string{"db"}); err != nil {
		t.Fatalf("SetGroupMembers: %v", err)
	}

	appGot := readServiceGroups(t, path, "app")
	if !slices.Equal(appGot, []string{"extra"}) {
		t.Errorf("app groups = %v, want [extra]", appGot)
	}
	if !hasGroupsKey(t, path, "app") {
		t.Error("app's profiles key should survive (still has extra)")
	}

	dbGot := readServiceGroups(t, path, "db")
	if !slices.Equal(dbGot, []string{"core"}) {
		t.Errorf("db groups = %v, want [core]", dbGot)
	}
}

func TestSetGroupMembers_Idempotent(t *testing.T) {
	path := writeFixture(t, baseFixture)

	// Reconciling to the current membership must be a no-op.
	if err := SetGroupMembers(path, "core", []string{"app", "db"}); err != nil {
		t.Fatalf("SetGroupMembers: %v", err)
	}

	if !slices.Equal(readServiceGroups(t, path, "app"), []string{"core"}) {
		t.Errorf("app groups changed by an idempotent reconcile")
	}
	if !slices.Equal(readServiceGroups(t, path, "db"), []string{"core"}) {
		t.Errorf("db groups changed by an idempotent reconcile")
	}
	if hasGroupsKey(t, path, "cache") {
		t.Error("cache gained a profiles key from an idempotent reconcile")
	}
}

func TestSetGroupMembers_PreservesComments(t *testing.T) {
	// The reconciliation is a node-tree edit, not a full re-encode from
	// structs, so comments on untouched lines must ride through.
	path := writeFixture(t, baseFixture)

	if err := SetGroupMembers(path, "core", []string{"app", "cache"}); err != nil {
		t.Fatalf("SetGroupMembers: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading result file: %v", err)
	}

	// app's # core services comment is attached to its profiles key, which
	// is preserved (app keeps core). Verify it survives the round trip.
	if !strings.Contains(string(raw), "core services") {
		t.Errorf("expected the # core services comment to survive, got:\n%s", raw)
	}
}

func TestRenameGroupTag_RetagsEveryCarrier(t *testing.T) {
	path := writeFixture(t, baseFixture)

	if err := RenameGroupTag(path, "core", "core2"); err != nil {
		t.Fatalf("RenameGroupTag: %v", err)
	}

	for _, service := range []string{"app", "db"} {
		if got := readServiceGroups(t, path, service); !slices.Equal(got, []string{"core2"}) {
			t.Errorf("%s groups = %v, want [core2]", service, got)
		}
	}

	// A service that never carried the tag must be untouched and must not
	// gain a profiles key.
	if hasGroupsKey(t, path, "cache") {
		t.Error("cache gained a profiles key from a rename that did not involve it")
	}
}

func TestRenameGroupTag_PreservesOtherProfiles(t *testing.T) {
	path := writeFixture(t, multiTagFixture)

	if err := RenameGroupTag(path, "core", "core2"); err != nil {
		t.Fatalf("RenameGroupTag: %v", err)
	}

	appGot := readServiceGroups(t, path, "app")
	want := []string{"core2", "extra"}
	if !slices.Equal(appGot, want) {
		t.Errorf("app groups = %v, want %v (extra must survive the rename)", appGot, want)
	}

	dbGot := readServiceGroups(t, path, "db")
	if !slices.Equal(dbGot, []string{"core2"}) {
		t.Errorf("db groups = %v, want [core2]", dbGot)
	}
}

func TestRenameGroupTag_PreservesComments(t *testing.T) {
	path := writeFixture(t, baseFixture)

	if err := RenameGroupTag(path, "core", "core2"); err != nil {
		t.Fatalf("RenameGroupTag: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading result file: %v", err)
	}

	if !strings.Contains(string(raw), "core services") {
		t.Errorf("expected the # core services comment to survive, got:\n%s", raw)
	}
}

func TestRenameGroupTag_NotFoundLeavesFileUntouched(t *testing.T) {
	path := writeFixture(t, baseFixture)

	err := RenameGroupTag(path, "nope", "core2")
	if err == nil {
		t.Fatal("expected an error when no service carries the group")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading result file: %v", err)
	}

	if string(raw) != baseFixture {
		t.Errorf("failed rename modified the file:\n%s", raw)
	}
}

func TestRenameGroupTag_MergesIntoExistingName(t *testing.T) {
	// Renaming onto a name some other service already carries merges the
	// tags; uniqueness is the modal's job, not the writer's - the same split
	// as AddGroupTag. The result is a duplicate entry on app, which docker
	// treats as one profile and the UI dedups on load.
	path := writeFixture(t, multiTagFixture)

	if err := RenameGroupTag(path, "extra", "core"); err != nil {
		t.Fatalf("RenameGroupTag: %v", err)
	}

	appGot := readServiceGroups(t, path, "app")
	want := []string{"core", "core"}
	if !slices.Equal(appGot, want) {
		t.Errorf("app groups = %v, want %v (merge, no dedup at the writer)", appGot, want)
	}
}
