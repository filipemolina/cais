package utils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
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

// tlsClientFor returns a client that trusts the test server's own
// certificate, so the check's TLS requirement (D4: HTTPS only) is exercised
// in tests rather than bypassed with an http server.
func tlsClientFor(t *testing.T, srv *httptest.Server) *http.Client {
	t.Helper()

	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())

	return &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}}}
}

// The flow is the one R1 measured working against Docker Hub, ghcr.io and
// lscr.io: a manifest request without a token is answered 401 with a
// WWW-Authenticate header naming the token realm; the token comes from that
// realm; the same manifest URL is retried with the bearer. Assertions pin
// each step, so a client that skips the token or guesses the realm fails
// here rather than silently as Unknown.
func TestBearerFlow(t *testing.T) {
	var manifestAuths []string
	var tokenQueries []string

	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		tokenQueries = append(tokenQueries, r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": "good"})
	})
	mux.HandleFunc("/v2/library/test/manifests/latest", func(w http.ResponseWriter, r *http.Request) {
		manifestAuths = append(manifestAuths, r.Header.Get("Authorization"))
		if manifestAuths[len(manifestAuths)-1] != "Bearer good" {
			w.Header().Set("WWW-Authenticate", `Bearer realm="`+srvRealmHost(r)+`/token",service="testsrv",scope="repository:library/test:pull"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Docker-Content-Digest", "sha256:858f009f9709ce576febc734aa78b8f6d624b82571f9ddb6bda4377c833b3499")
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	digest := fetchRemoteDigest(tlsClientFor(t, srv), ImageRef{Host: "127.0.0.1:" + portOf(t, srv.URL), Repo: "library/test", Tag: "latest"}, time.Second)

	want := "sha256:858f009f9709ce576febc734aa78b8f6d624b82571f9ddb6bda4377c833b3499"
	if digest != want {
		t.Errorf("digest = %q, want %q", digest, want)
	}

	if len(manifestAuths) != 2 || manifestAuths[0] != "" || manifestAuths[1] != "Bearer good" {
		t.Errorf("manifest requests carried %v, want [\"\" then the bearer from the token]", manifestAuths)
	}
	if len(tokenQueries) != 1 {
		t.Fatalf("token endpoint hit %d times, want 1", len(tokenQueries))
	}
	if !strings.Contains(tokenQueries[0], "service=testsrv") || !strings.Contains(tokenQueries[0], "scope=repository%3Alibrary%2Ftest%3Apull") {
		t.Errorf("token request query = %q, want service and scope carried through", tokenQueries[0])
	}
}

// A 401 on the retried attempt is that repo's Unknown — never Stale. This is
// the private-registry shape (D4): the anonymous token is not good enough,
// and the check must fall silent rather than report an update it could not
// actually verify.
func TestBearerFlowRetryStillUnauthorized(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"token": "anonymous"})
	})
	mux.HandleFunc("/v2/library/test/manifests/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="`+srvRealmHost(r)+`/token"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	digest := fetchRemoteDigest(tlsClientFor(t, srv), ImageRef{Host: "127.0.0.1:" + portOf(t, srv.URL), Repo: "library/test", Tag: "latest"}, time.Second)

	if digest != "" {
		t.Errorf("digest = %q, want \"\" — a 401 after the retry is Unknown, never an answer", digest)
	}
}

func TestManifestDirectAnswer(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Docker-Content-Digest", "sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	digest := fetchRemoteDigest(tlsClientFor(t, srv), ImageRef{Host: "127.0.0.1:" + portOf(t, srv.URL), Repo: "library/test", Tag: "latest"}, time.Second)

	if digest != "sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6" {
		t.Errorf("digest = %q, want the header's value", digest)
	}
}

// A 200 without the header is no answer at all: every classification rides
// on the header, and accepting the body would re-open the exact trap R2
// records for digest-taking shortcuts.
func TestManifestWithoutDigestHeaderIsNoAnswer(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if digest := fetchRemoteDigest(tlsClientFor(t, srv), ImageRef{Host: "127.0.0.1:" + portOf(t, srv.URL), Repo: "library/test", Tag: "latest"}, time.Second); digest != "" {
		t.Errorf("digest = %q, want \"\"", digest)
	}
}

func TestRemoteFailureIsSilent(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()

	ref := ImageRef{Host: "127.0.0.1:" + portOf(t, srv.URL), Repo: "library/test", Tag: "latest"}

	if digest := fetchRemoteDigest(tlsClientFor(t, srv), ref, time.Second); digest != "" {
		t.Errorf("5xx digest = %q, want \"\"", digest)
	}

	closed := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closed.Close()
	ref.Host = "127.0.0.1:" + portOf(t, closed.URL)
	if digest := fetchRemoteDigest(tlsClientFor(t, closed), ref, time.Second); digest != "" {
		t.Errorf("unreachable registry digest = %q, want \"\"", digest)
	}
}

// A realm that is not HTTPS is refused before any request is sent to it.
func TestBearerTokenRealmNotHTTPS(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/library/test/manifests/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="http://elsewhere/token"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	if digest := fetchRemoteDigest(tlsClientFor(t, srv), ImageRef{Host: "127.0.0.1:" + portOf(t, srv.URL), Repo: "library/test", Tag: "latest"}, time.Second); digest != "" {
		t.Errorf("digest = %q, want \"\"", digest)
	}
}

func TestBearerTokenWithoutTokenInResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"unrelated": "value"})
	})
	mux.HandleFunc("/v2/library/test/manifests/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="`+srvRealmHost(r)+`/token"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	if digest := fetchRemoteDigest(tlsClientFor(t, srv), ImageRef{Host: "127.0.0.1:" + portOf(t, srv.URL), Repo: "library/test", Tag: "latest"}, time.Second); digest != "" {
		t.Errorf("digest = %q, want \"\"", digest)
	}
}

func TestParseAuthChallenge(t *testing.T) {
	challenge := parseAuthChallenge(`Bearer realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:library/redis:pull"`)
	if challenge == nil {
		t.Fatal("parseAuthChallenge returned nil for a Bearer header")
	}
	if challenge.realm != "https://auth.docker.io/token" {
		t.Errorf("realm = %q, want %q", challenge.realm, "https://auth.docker.io/token")
	}
	if challenge.service != "registry.docker.io" {
		t.Errorf("service = %q, want %q", challenge.service, "registry.docker.io")
	}
	if challenge.scope != "repository:library/redis:pull" {
		t.Errorf("scope = %q, want %q", challenge.scope, "repository:library/redis:pull")
	}

	for _, header := range []string{"", `Basic realm="https://x/token"`, `Bearer service="x"`} {
		if challenge := parseAuthChallenge(header); challenge != nil {
			t.Errorf("parseAuthChallenge(%q) = %v, want nil", header, challenge)
		}
	}
}

// The pool level: refs two services share are deduped into one registry
// call, and the map files each unique ref under its ImageRefKey.
func TestRemoteImageDigestsDedupesAndFilesByRefKey(t *testing.T) {
	original := newRegistryClient
	t.Cleanup(func() { newRegistryClient = original })

	var manifests int
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/library/test/manifests/", func(w http.ResponseWriter, r *http.Request) {
		manifests++
		w.Header().Set("Docker-Content-Digest", "sha256:858f009f9709ce576febc734aa78b8f6d624b82571f9ddb6bda4377c833b3499")
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	client := tlsClientFor(t, srv)
	newRegistryClient = func() *http.Client { return client }

	host := "127.0.0.1:" + portOf(t, srv.URL)
	pinned, _ := ParseImageRef(host + "/library/test:latest")
	implied, _ := ParseImageRef(host + "/library/test") // same ref, tag defaulted

	updates := RemoteImageDigests([]ImageRef{pinned, implied, implied}, time.Second)

	if manifests != 1 {
		t.Errorf("registry hit %d times, want 1 — duplicate refs share the call", manifests)
	}

	key := ImageRefKey(ImageRef{Host: host, Repo: "library/test", Tag: "latest"})
	got, ok := updates[key]
	if !ok {
		t.Fatalf("updates has no entry for %q (has %v)", key, updates)
	}
	if got.RemoteDigest != "sha256:858f009f9709ce576febc734aa78b8f6d624b82571f9ddb6bda4377c833b3499" {
		t.Errorf("RemoteDigest = %q, want the served digest", got.RemoteDigest)
	}
	if got.Repo != "library/test" {
		t.Errorf("Repo = %q, want %q", got.Repo, "library/test")
	}
}

// srvRealmHost is the https://host:port the test server answers on, as a
// WWW-Authenticate realm would name it.
func srvRealmHost(r *http.Request) string {
	return "https://" + r.Host
}

// portOf pulls the port out of an httptest server's URL.
func portOf(t *testing.T, url string) string {
	t.Helper()

	i := strings.LastIndex(url, ":")
	if i < 0 {
		t.Fatalf("no port in %q", url)
	}

	return url[i+1:]
}
