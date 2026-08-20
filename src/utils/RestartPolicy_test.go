package utils

import (
	"strings"
	"testing"
)

func TestNextRestartPolicyCyclesThroughAllFourValues(t *testing.T) {
	cases := []struct {
		current string
		want    string
	}{
		{"", "on-failure"},
		{"on-failure", "unless-stopped"},
		{"unless-stopped", "always"},
		{"always", ""},
		// "no" is the unset state's explicit spelling - treated the same as "".
		{"no", "on-failure"},
		// An unrecognised value resets to the start of the cycle.
		{"bogus", ""},
	}

	for _, tc := range cases {
		t.Run(tc.current, func(t *testing.T) {
			if got := NextRestartPolicy(tc.current); got != tc.want {
				t.Errorf("NextRestartPolicy(%q) = %q, want %q", tc.current, got, tc.want)
			}
		})
	}
}

func TestSetRestartPolicyInsertsANewKey(t *testing.T) {
	path := writeFixture(t, "services:\n  web:\n    image: nginx:alpine\n")

	if err := SetRestartPolicy(path, "web", "unless-stopped"); err != nil {
		t.Fatalf("SetRestartPolicy: %v", err)
	}

	after := readFile(t, path)
	if !strings.Contains(after, "restart: unless-stopped") {
		t.Errorf("missing restart: unless-stopped, got:\n%s", after)
	}
}

func TestSetRestartPolicyReplacesAnExistingValue(t *testing.T) {
	path := writeFixture(t, "services:\n  web:\n    image: nginx:alpine\n    restart: always\n")

	if err := SetRestartPolicy(path, "web", "on-failure"); err != nil {
		t.Fatalf("SetRestartPolicy: %v", err)
	}

	after := readFile(t, path)
	if strings.Count(after, "restart:") != 1 {
		t.Fatalf("expected exactly one restart: key, got:\n%s", after)
	}
	if !strings.Contains(after, "restart: on-failure") {
		t.Errorf("old value was not replaced, got:\n%s", after)
	}
}

// An empty policy removes the key rather than writing restart: "" - compose
// has no such value, and a dropped key is what "no policy" looks like on
// disk.
func TestSetRestartPolicyEmptyRemovesTheKey(t *testing.T) {
	path := writeFixture(t, "services:\n  web:\n    image: nginx:alpine\n    restart: always\n")

	if err := SetRestartPolicy(path, "web", ""); err != nil {
		t.Fatalf("SetRestartPolicy: %v", err)
	}

	after := readFile(t, path)
	if strings.Contains(after, "restart:") {
		t.Errorf("restart: key should have been removed, got:\n%s", after)
	}
}

func TestSetRestartPolicyEmptyOnAServiceWithNoKeyIsANoop(t *testing.T) {
	path := writeFixture(t, "services:\n  web:\n    image: nginx:alpine\n")
	before := readFile(t, path)

	if err := SetRestartPolicy(path, "web", ""); err != nil {
		t.Fatalf("SetRestartPolicy: %v", err)
	}

	if after := readFile(t, path); after != before {
		t.Errorf("removing an absent restart: key changed the file:\n%s", after)
	}
}

// Other services, their comments, and their own key order are untouched.
func TestSetRestartPolicyLeavesOtherServicesUntouched(t *testing.T) {
	path := writeFixture(t, `services:
  db:
    image: postgres:16
  web:
    image: nginx:alpine # the frontend
    ports:
      - "8080:80"
`)

	if err := SetRestartPolicy(path, "db", "always"); err != nil {
		t.Fatalf("SetRestartPolicy: %v", err)
	}

	after := readFile(t, path)
	for _, untouched := range []string{"# the frontend", `"8080:80"`} {
		if !strings.Contains(after, untouched) {
			t.Errorf("setting db's restart policy lost %q, got:\n%s", untouched, after)
		}
	}
}

func TestSetRestartPolicyUnknownServiceIsRejected(t *testing.T) {
	path := writeFixture(t, "services:\n  db:\n    image: postgres:16\n")

	if err := SetRestartPolicy(path, "nope", "always"); err == nil {
		t.Fatal("expected an error for an unknown service")
	}
}
