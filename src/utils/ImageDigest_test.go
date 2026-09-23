package utils

import (
	"context"
	"os"
	"os/exec"
	"testing"
)

func TestParseImageRef(t *testing.T) {
	tests := []struct {
		input string
		host  string
		repo  string
		tag   string
	}{
		{"redis:7-alpine", "docker.io", "library/redis", "7-alpine"},
		{"prom/prometheus:latest", "docker.io", "prom/prometheus", "latest"},
		{"ghcr.io/paperless-ngx/paperless-ngx:latest", "ghcr.io", "paperless-ngx/paperless-ngx", "latest"},
		{"lscr.io/linuxserver/sonarr:latest", "lscr.io", "linuxserver/sonarr", "latest"},
		{"registry.example.com:5000/app:v1", "registry.example.com:5000", "app", "v1"},
		{"alpine", "docker.io", "library/alpine", "latest"},
		{"redis@sha256:6ab0b6e7381779332f97b8ca76193e45b0756f38d4c0dcda72dbb3c32061ab99", "docker.io", "library/redis", "latest"},
		{"redis:7-alpine@sha256:6ab0b6e7381779332f97b8ca76193e45b0756f38d4c0dcda72dbb3c32061ab99", "docker.io", "library/redis", "7-alpine"},
	}

	for _, tc := range tests {
		ref, err := ParseImageRef(tc.input)
		if err != nil {
			t.Errorf("ParseImageRef(%q): %v", tc.input, err)
			continue
		}

		if ref.Host != tc.host || ref.Repo != tc.repo || ref.Tag != tc.tag {
			t.Errorf("ParseImageRef(%q) = host %q repo %q tag %q, want host %q repo %q tag %q",
				tc.input, ref.Host, ref.Repo, ref.Tag, tc.host, tc.repo, tc.tag)
		}
	}
}

func TestParseImageRefDigest(t *testing.T) {
	const digest = "sha256:6ab0b6e7381779332f97b8ca76193e45b0756f38d4c0dcda72dbb3c32061ab99"

	ref, err := ParseImageRef("redis:7-alpine@" + digest)
	if err != nil {
		t.Fatalf("ParseImageRef: %v", err)
	}

	if ref.Digest != digest {
		t.Errorf("digest = %q, want %q", ref.Digest, digest)
	}

	plain, err := ParseImageRef("redis:7-alpine")
	if err != nil {
		t.Fatalf("ParseImageRef: %v", err)
	}

	if plain.Digest != "" {
		t.Errorf("digest = %q, want empty", plain.Digest)
	}
}

func TestParseImageRefErrors(t *testing.T) {
	tests := []string{
		"",
		"   ",
		"@sha256:6ab0b6e7381779332f97b8ca76193e45b0756f38d4c0dcda72dbb3c32061ab99",
		"redis:",
		"redis@",
		"redis@sha256:",
		"redis@notadigest",
	}

	for _, input := range tests {
		if _, err := ParseImageRef(input); err == nil {
			t.Errorf("ParseImageRef(%q) = nil error, want an error", input)
		}
	}
}

func TestIsPinnedToDigest(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"redis@sha256:6ab0b6e7381779332f97b8ca76193e45b0756f38d4c0dcda72dbb3c32061ab99", true},
		{"alpine", false},
		{"redis:7-alpine", false},
		// A tagged digest pin is not a pure pin: its tag can move and the
		// check must still run against it (D4), or the feature is one-shot.
		{"redis:7-alpine@sha256:6ab0b6e7381779332f97b8ca76193e45b0756f38d4c0dcda72dbb3c32061ab99", false},
		{"redis@", false},
	}

	for _, tc := range tests {
		if got := IsPinnedToDigest(tc.input); got != tc.want {
			t.Errorf("IsPinnedToDigest(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

// The last two rows are the ones that matter: they are D3, the rule that a
// failed check must never read as an available update or as verified
// up-to-date. A flaky network producing a wall of update glyphs would kill
// trust in the feature faster than any missed update.
func TestClassifyImageUpdate(t *testing.T) {
	tests := []struct {
		local  string
		remote string
		want   ImageUpdateState
	}{
		{"sha256:a", "sha256:a", ImageUpToDate},
		{"sha256:a", "sha256:b", ImageStale},
		{"", "sha256:b", ImageUnknown},
		{"sha256:a", "", ImageUnknown},
		{"", "", ImageUnknown},
	}

	for _, tc := range tests {
		if got := ClassifyImageUpdate(tc.local, tc.remote); got != tc.want {
			t.Errorf("ClassifyImageUpdate(%q, %q) = %v, want %v", tc.local, tc.remote, got, tc.want)
		}
	}
}

// The outputs are captured verbatim from `docker image inspect <ref>
// --format '{{json .RepoDigests}}'` on 2026-09-23, engine 29.8.0. The
// familiar-name forms are the R4 trap: the daemon writes alpine's entry as
// "alpine@…" — no docker.io, no library/ — while a ghcr pull carries the
// host, so neither side of the compare is usable raw.
func TestDigestForRepo(t *testing.T) {
	const alpineOutput = `["alpine@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6"]`
	const redisOutput = `["redis@sha256:6ab0b6e7381779332f97b8ca76193e45b0756f38d4c0dcda72dbb3c32061ab99"]`
	const ghcrOutput = `["ghcr.io/fallenbagel/jellyseerr@sha256:9cc9e9ee6cd5cf5a23feb45c37742ba34cfd6314d81d259cddb373a97ac92cdd"]`

	tests := []struct {
		name   string
		output string
		repo   string
		want   string
	}{
		{"official image, familiar name", alpineOutput, "library/alpine", "sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6"},
		{"official image, tag form", redisOutput, "library/redis", "sha256:6ab0b6e7381779332f97b8ca76193e45b0756f38d4c0dcda72dbb3c32061ab99"},
		{"host-prefixed repo", ghcrOutput, "fallenbagel/jellyseerr", "sha256:9cc9e9ee6cd5cf5a23feb45c37742ba34cfd6314d81d259cddb373a97ac92cdd"},
		{"repo mismatch filters the entry", alpineOutput, "library/redis", ""},
		{"host mismatch filters the entry", ghcrOutput, "library/alpine", ""},
		// A locally built image prints [] — empty result, no error, no panic (R4).
		{"empty array", `[]`, "library/alpine", ""},
		{"garbage output", "Error: something went wrong", "library/alpine", ""},
		{"empty output", "", "library/alpine", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := digestForRepo(tc.output, tc.repo); got != tc.want {
				t.Errorf("digestForRepo(%q, %q) = %q, want %q", tc.output, tc.repo, got, tc.want)
			}
		})
	}
}

// An image pulled through two repos carries one entry per repo; the entry
// for this ref's repo is picked, not "newest wins" — the order is the
// daemon's, not a freshness signal.
func TestDigestForRepoMultiEntry(t *testing.T) {
	const multi = `["alpine@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6",` +
		`"ghcr.io/fallenbagel/jellyseerr@sha256:9cc9e9ee6cd5cf5a23feb45c37742ba34cfd6314d81d259cddb373a97ac92cdd"]`

	if got := digestForRepo(multi, "fallenbagel/jellyseerr"); got != "sha256:9cc9e9ee6cd5cf5a23feb45c37742ba34cfd6314d81d259cddb373a97ac92cdd" {
		t.Errorf("digestForRepo multi-entry = %q, want the ghcr entry", got)
	}
}

// The exec half runs the inspect once per ref, not batched, and prints what
// it found. The fake binary is the same seam DockerComposePs's test uses.
func TestLocalImageDigestsInspectsEachRef(t *testing.T) {
	original := dockerCommandContext
	t.Cleanup(func() { dockerCommandContext = original })

	var calls [][]string
	dockerCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		calls = append(calls, append([]string{name}, args...))
		script := "#!/bin/sh\n" +
			`echo '["alpine@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6"]'` + "\n"
		return fakeShellScript(t, script)
	}

	got, err := LocalImageDigests([]string{"alpine:latest", "alpine:latest", "redis:7-alpine"})
	if err != nil {
		t.Fatalf("LocalImageDigests: %v", err)
	}

	want := "sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6"
	if got["alpine:latest"] != want {
		t.Errorf("alpine:latest = %q, want %q", got["alpine:latest"], want)
	}
	// The fake prints alpine's entry for every call, but redis's repo filter
	// must not match an alpine entry.
	if got["redis:7-alpine"] != "" {
		t.Errorf("redis:7-alpine = %q, want \"\" (repo filter)", got["redis:7-alpine"])
	}

	if len(calls) != 2 {
		t.Errorf("inspect ran %d times, want 2 (one per unique ref)", len(calls))
	}
}

func TestLocalImageDigestsAbsentRefIsEmpty(t *testing.T) {
	original := dockerCommandContext
	t.Cleanup(func() { dockerCommandContext = original })

	// docker exits 1 and prints "Error: No such image: <ref>" for a ref
	// that is not in the local store; that is the "" case, not an error.
	script := "#!/bin/sh\necho 'Error: No such image: nosuchthing' >&2\nexit 1\n"
	dockerCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return fakeShellScript(t, script)
	}

	got, err := LocalImageDigests([]string{"nosuchthing:latest"})
	if err != nil {
		t.Fatalf("LocalImageDigests: %v", err)
	}

	if got["nosuchthing:latest"] != "" {
		t.Errorf("absent ref = %q, want \"\"", got["nosuchthing:latest"])
	}
}

func TestLocalImageDigestsDaemonDownIsAnError(t *testing.T) {
	original := dockerCommandContext
	t.Cleanup(func() { dockerCommandContext = original })

	script := "#!/bin/sh\necho 'Cannot connect to the Docker daemon at unix:///var/run/docker.sock' >&2\nexit 1\n"
	dockerCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return fakeShellScript(t, script)
	}

	if _, err := LocalImageDigests([]string{"alpine:latest"}); err == nil {
		t.Error("daemon-down inspect returned nil error, want an error the command swallows")
	}
}

func TestLocalImageDigestsUnparseableRefSkipsExec(t *testing.T) {
	original := dockerCommandContext
	t.Cleanup(func() { dockerCommandContext = original })

	ran := false
	dockerCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		ran = true
		return exec.Command("true")
	}

	got, err := LocalImageDigests([]string{"redis@sha256:nothex!"})
	if err != nil {
		t.Fatalf("LocalImageDigests: %v", err)
	}

	if ran {
		t.Error("an unparseable ref still ran an inspect")
	}
	if got["redis@sha256:nothex!"] != "" {
		t.Errorf("unparseable ref = %q, want \"\"", got["redis@sha256:nothex!"])
	}
}

// fakeShellScript writes a shell script standing in for the docker binary,
// the way DockerCommand.go's seam intends: the test can exercise the code
// that reads docker's output without docker being installed.
func fakeShellScript(t *testing.T, script string) *exec.Cmd {
	t.Helper()

	path := t.TempDir() + "/fake-docker"
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatalf("write fake docker: %v", err)
	}

	return exec.Command(path)
}
