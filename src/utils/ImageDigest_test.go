package utils

import "testing"

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
