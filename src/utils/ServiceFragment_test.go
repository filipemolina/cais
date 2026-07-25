package utils

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fragmentFixture deliberately carries comments, a blank line between
// services, and a quoted port, because preserving all three is the point of
// editing a fragment rather than regenerating the file from a form.
const fragmentFixture = `services:
  app:
    image: nginx:alpine
    profiles: ["core"] # core services
    ports:
      - "8085:80"

  db:
    image: postgres:alpine # the database
    profiles: ["core"]

  cache:
    image: redis:alpine
`

func TestExtractServiceFragment(t *testing.T) {
	path := writeFixture(t, fragmentFixture)

	fragment, err := ExtractServiceFragment(path, "app")
	if err != nil {
		t.Fatalf("ExtractServiceFragment: %v", err)
	}

	got := string(fragment)

	// The service name is the top-level key, so the fragment reads the way
	// it does in the file.
	if !strings.HasPrefix(got, "app:") {
		t.Errorf("fragment should start with the service key, got:\n%s", got)
	}
	if !strings.Contains(got, "# core services") {
		t.Errorf("fragment lost the service's comment, got:\n%s", got)
	}
	if !strings.Contains(got, `"8085:80"`) {
		t.Errorf("fragment lost the quoting on the port, got:\n%s", got)
	}
	for _, other := range []string{"db:", "cache:", "postgres", "redis"} {
		if strings.Contains(got, other) {
			t.Errorf("fragment leaked %q from a neighbouring service, got:\n%s", other, got)
		}
	}
}

func TestExtractServiceFragmentUnknownService(t *testing.T) {
	path := writeFixture(t, fragmentFixture)

	if _, err := ExtractServiceFragment(path, "nope"); err == nil {
		t.Fatal("extracting an unknown service should fail")
	}
}

// A fragment that goes out and comes back untouched should change as little
// as possible. It is not byte-identical: the blank line at the end of the
// edited service is lost, because it attaches to the last node inside that
// service and goes out with the subtree being replaced. Everything else -
// comments, quoting, key order, and the blank lines around every other
// service - survives.
func TestApplyServiceFragmentRoundTripsAlmostUnchanged(t *testing.T) {
	path := writeFixture(t, fragmentFixture)
	before := readFile(t, path)

	fragment, err := ExtractServiceFragment(path, "app")
	if err != nil {
		t.Fatalf("ExtractServiceFragment: %v", err)
	}

	if err := ApplyServiceFragment(path, "app", fragment); err != nil {
		t.Fatalf("ApplyServiceFragment: %v", err)
	}

	after := readFile(t, path)

	// The one documented difference, and nothing else.
	if want := strings.Replace(before, "- \"8085:80\"\n\n", "- \"8085:80\"\n", 1); after != want {
		t.Errorf("a round trip changed more than the trailing blank line:\n--- want ---\n%s\n--- got ---\n%s", want, after)
	}

	// The blank line before an untouched service is not collateral.
	if !strings.Contains(after, "profiles: [\"core\"]\n\n  cache:") {
		t.Errorf("the blank line before an untouched service was lost:\n%s", after)
	}
}

// The blank-line preservation is not specific to editing: every write goes
// through the same encoder, and tagging a group used to strip the spacing
// out of the whole file.
func TestTaggingAGroupKeepsBlankLines(t *testing.T) {
	path := writeFixture(t, fragmentFixture)

	if err := AddGroupTag(path, "extra", []string{"cache"}); err != nil {
		t.Fatalf("AddGroupTag: %v", err)
	}

	after := readFile(t, path)

	for _, gap := range []string{"- \"8085:80\"\n\n  db:", "profiles: [\"core\"]\n\n  cache:"} {
		if !strings.Contains(after, gap) {
			t.Errorf("tagging a group closed up the spacing around %q:\n%s", gap, after)
		}
	}
}

func TestApplyServiceFragmentWritesTheEdit(t *testing.T) {
	path := writeFixture(t, fragmentFixture)

	edited := `app:
  image: nginx:1.28
  profiles: ["core"] # core services
  ports:
    - "8085:80"
    - "8443:443"
`

	if err := ApplyServiceFragment(path, "app", []byte(edited)); err != nil {
		t.Fatalf("ApplyServiceFragment: %v", err)
	}

	after := readFile(t, path)

	if !strings.Contains(after, "nginx:1.28") {
		t.Errorf("the edit was not written, got:\n%s", after)
	}
	if !strings.Contains(after, `"8443:443"`) {
		t.Errorf("the added port was not written, got:\n%s", after)
	}

	// Everything the user did not touch has to survive.
	for _, untouched := range []string{
		"# core services",
		"postgres:alpine # the database",
		"redis:alpine",
	} {
		if !strings.Contains(after, untouched) {
			t.Errorf("editing one service lost %q, got:\n%s", untouched, after)
		}
	}
}

// Every rejection has to leave the file byte-identical: the user's edit is
// refused, not half-applied.
func TestApplyServiceFragmentRejections(t *testing.T) {
	cases := []struct {
		name     string
		fragment string
		wantErr  string
	}{
		{
			name:     "unparseable YAML",
			fragment: "app:\n  image: [unclosed\n",
			wantErr:  "not valid YAML",
		},
		{
			name:     "renamed service",
			fragment: "renamed:\n  image: nginx:alpine\n",
			wantErr:  "renaming a service is not supported",
		},
		{
			name:     "more than one service",
			fragment: "app:\n  image: nginx:alpine\nextra:\n  image: redis\n",
			wantErr:  "exactly one service",
		},
		{
			name:     "not a mapping",
			fragment: "app: nginx:alpine\n",
			wantErr:  "block of keys",
		},
		{
			name:     "empty",
			fragment: "",
			wantErr:  "empty",
		},
		{
			// Parses as YAML, and is shaped like a service, but compose
			// will not have it. This is the check a plain YAML parse misses.
			name:     "valid YAML that is not valid compose",
			fragment: "app:\n  image: nginx:alpine\n  ports:\n    - target: not-a-number\n",
			wantErr:  "",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			path := writeFixture(t, fragmentFixture)
			before := readFile(t, path)

			err := ApplyServiceFragment(path, "app", []byte(testCase.fragment))
			if err == nil {
				t.Fatal("expected the edit to be rejected")
			}
			if testCase.wantErr != "" && !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("error %q does not mention %q", err, testCase.wantErr)
			}

			if after := readFile(t, path); after != before {
				t.Errorf("a rejected edit changed the file:\n%s", after)
			}
		})
	}
}

func TestApplyServiceFragmentRenameIsIdentifiable(t *testing.T) {
	path := writeFixture(t, fragmentFixture)

	err := ApplyServiceFragment(path, "app", []byte("renamed:\n  image: nginx:alpine\n"))
	if !errors.Is(err, ErrServiceRenamed) {
		t.Errorf("a rename should be identifiable as ErrServiceRenamed, got %v", err)
	}
}

func TestValidateComposeCandidateLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()

	if err := ValidateComposeCandidate(dir, []byte(fragmentFixture)); err != nil {
		t.Fatalf("a valid compose file was rejected: %v", err)
	}
	if err := ValidateComposeCandidate(dir, []byte("services: 12\n")); err == nil {
		t.Fatal("an invalid compose file was accepted")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	if len(entries) != 0 {
		t.Errorf("validation left %d file(s) behind: %v", len(entries), entries)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", filepath.Base(path), err)
	}

	return string(contents)
}
