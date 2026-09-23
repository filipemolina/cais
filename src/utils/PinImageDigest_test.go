package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The digest is a real one, captured from the Hub's Docker-Content-Digest
// for redis:7-alpine on 2026-09-23 (R1); a bare hex string is the malformed
// case the function must refuse.
const pinDigest = "sha256:858f009f9709ce576febc734aa78b8f6d624b82571f9ddb6bda4377c833b3499"

func TestPinImageDigestAppendsThePin(t *testing.T) {
	path := writeFixture(t, fragmentFixture)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	if err := PinImageDigest(path, "cache", pinDigest); err != nil {
		t.Fatalf("PinImageDigest: %v", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading pinned file: %v", err)
	}
	got := string(after)

	if !strings.Contains(got, "image: redis:alpine@"+pinDigest) {
		t.Errorf("pinned file does not carry the pin, got:\n%s", got)
	}
	// The user's own writing survives the edit — comments attached to other
	// keys and the untouched services.
	if !strings.Contains(got, "postgres:alpine # the database") {
		t.Errorf("pin edit disturbed a neighbour's comment, got:\n%s", got)
	}
	if !strings.Contains(got, "profiles: [\"core\"] # core services") {
		t.Errorf("pin edit lost a comment, got:\n%s", got)
	}

	// The snapshot is the rollback story: a .bak carrying the pre-pin bytes
	// must exist, taken by the same atomic write every other edit uses —
	// no special backup code.
	bak := newestBackupFor(t, path)
	if bak == nil || string(bak) != string(before) {
		t.Errorf("the pre-write snapshot does not match the pre-pin file")
	}
}

func TestPinImageDigestReplacesAnExistingDigest(t *testing.T) {
	fixture := `services:
  app:
    image: redis:7-alpine@sha256:6ab0b6e7381779332f97b8ca76193e45b0756f38d4c0dcda72dbb3c32061ab99
`
	path := writeFixture(t, fixture)

	if err := PinImageDigest(path, "app", pinDigest); err != nil {
		t.Fatalf("PinImageDigest: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading pinned file: %v", err)
	}

	// The re-pin swaps the digest; it must not stack a second @ clause.
	if !strings.Contains(string(got), "image: redis:7-alpine@"+pinDigest) {
		t.Errorf("re-pin did not replace the digest, got:\n%s", got)
	}
	if strings.Count(string(got), "@") != 1 {
		t.Errorf("re-pin left more than one @ in the reference, got:\n%s", got)
	}
}

func TestPinImageDigestRejections(t *testing.T) {
	tests := []struct {
		name    string
		service string
		digest  string
	}{
		{"bare hex digest", "cache", "858f009f9709ce576febc734aa78b8f6d624b82571f9ddb6bda4377c833b3499"},
		{"digest without encoded part", "cache", "sha256:"},
		{"unknown service", "nosuch", pinDigest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := writeFixture(t, fragmentFixture)
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading fixture: %v", err)
			}

			if err := PinImageDigest(path, tc.service, tc.digest); err == nil {
				t.Fatalf("PinImageDigest(%s, %s) = nil error, want a refusal", tc.service, tc.digest)
			}

			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading file after refusal: %v", err)
			}
			// A refused pin leaves the file exactly as it was: the whole
			// point of validate-before-write.
			if string(after) != string(before) {
				t.Errorf("refused pin changed the file, got:\n%s", after)
			}
		})
	}
}

func TestPinImageDigestRefusesServiceWithoutImage(t *testing.T) {
	fixture := `services:
  app:
    build: ./app
`
	path := writeFixture(t, fixture)

	if err := PinImageDigest(path, "app", pinDigest); err == nil {
		t.Fatal("PinImageDigest = nil error, want a refusal for a service with no image: key")
	}
}

// newestBackupFor reads the most recent .bak entry the snapshot took for
// path, or nil when the store has none.
func newestBackupFor(t *testing.T, path string) []byte {
	t.Helper()

	dir := snapshotDirFor(path)
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		return nil
	}

	// Entries sort lexically by timestamp prefix and the newest is last.
	newest := entries[len(entries)-1]
	contents, err := os.ReadFile(filepath.Join(dir, newest.Name()))
	if err != nil {
		t.Fatalf("reading snapshot: %v", err)
	}

	return contents
}
