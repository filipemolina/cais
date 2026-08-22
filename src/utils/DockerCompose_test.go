package utils

import (
	"slices"
	"strings"
	"testing"
)

// A group action names its member services rather than requesting the
// profile. Measured on Compose v5.5.0 with a three-service file (core_a in
// profile core, untagged_b and untagged_c with no profiles: key):
//
//	$ docker compose --profile core up -d   -> started core_a, untagged_b AND untagged_c
//	$ docker compose up -d core_a           -> started ONLY core_a
//
// `--profile X <verb>` also activates every service with no profiles: key,
// so a group start/stop/restart/pull/remove reached services the user never
// selected. Naming the members scopes the command to exactly them, and
// auto-enables their profile on the way - so the flag is not needed alongside
// the names and must not appear in the args at all.
func TestGroupActionNamesItsMembersNotTheProfile(t *testing.T) {
	args, err := composeActionArgs("start", "core", true, "compose.yaml", []string{"core_a", "db"})
	if err != nil {
		t.Fatalf("composeActionArgs: %v", err)
	}

	want := []string{"compose", "--file", "compose.yaml", "up", "-d", "core_a", "db"}
	if !slices.Equal(args, want) {
		t.Errorf("args = %q, want %q", args, want)
	}
	if slices.Contains(args, "--profile") {
		t.Errorf("args contain --profile: %q", args)
	}
}

// A single-service action names that one service, the same shape it always
// had - the change is only in what a group action passes.
func TestSingleServiceActionNamesTheService(t *testing.T) {
	args, err := composeActionArgs("start", "web", false, "compose.yaml", nil)
	if err != nil {
		t.Fatalf("composeActionArgs: %v", err)
	}

	want := []string{"compose", "--file", "compose.yaml", "up", "-d", "web"}
	if !slices.Equal(args, want) {
		t.Errorf("args = %q, want %q", args, want)
	}
}

// Every verb maps to the right subcommand, with remove producing `rm -fs`
// rather than `down` (down would tear down the project's shared network).
func TestEveryActionVerbMapsToItsSubcommand(t *testing.T) {
	tests := []struct {
		action string
		want   []string
	}{
		{"start", []string{"up", "-d"}},
		{"stop", []string{"stop"}},
		{"restart", []string{"restart"}},
		{"pull", []string{"pull"}},
		{"remove", []string{"rm", "-fs"}},
	}

	for _, tc := range tests {
		args, err := composeActionArgs(tc.action, "web", false, "compose.yaml", nil)
		if err != nil {
			t.Fatalf("%s: composeActionArgs: %v", tc.action, err)
		}

		// The subcommand sits between the --file args and the target.
		got := args[3 : 3+len(tc.want)]
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s subcommand = %q, want %q (full args %q)", tc.action, got, tc.want, args)
		}
	}
}

// An empty member list is refused rather than run. `up -d` with no service
// names is "start the default set" - every service with no profiles: key -
// which is the very over-reach naming the members removed, only now with no
// group named at all. Measured on Compose v5.5.0:
//
//	$ docker compose up -d   (no names) -> started untagged_b and untagged_c
func TestEmptyGroupActionIsRefusedRatherThanRun(t *testing.T) {
	args, err := composeActionArgs("start", "core", true, "compose.yaml", nil)
	if err == nil {
		t.Fatal("expected an error for an empty member list")
	}
	if !strings.Contains(err.Error(), `group "core" has no services`) {
		t.Errorf("error = %q, want it to name the group", err)
	}
	if args != nil {
		t.Errorf("args = %q, want nil when refused", args)
	}
}

// An unknown action is a programming error, not a docker failure - it is
// refused before any command is built.
func TestUnknownActionIsRefused(t *testing.T) {
	args, err := composeActionArgs("explode", "web", false, "compose.yaml", nil)
	if err == nil || !strings.Contains(err.Error(), "unknown docker compose action") {
		t.Fatalf("err = %v, want the unknown-action error", err)
	}
	if args != nil {
		t.Errorf("args = %q, want nil", args)
	}
}
