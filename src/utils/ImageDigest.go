package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ImageUpdateState is what a check concluded about one service's image.
// The zero value is ImageUnknown: a failed check must never read as
// "update available", and must never read as "verified up to date".
// A flaky network or a private registry therefore renders as nothing at
// all, the same contract 03-drift-detection.md §D4 puts on drift.
type ImageUpdateState string

const (
	ImageUnknown  ImageUpdateState = "unknown"
	ImageUpToDate ImageUpdateState = "up-to-date"
	ImageStale    ImageUpdateState = "update-available"
)

// ImageUpdate is one service's answer. Digests carry the sha256:… form; the
// UI shortens them for display.
type ImageUpdate struct {
	Service      string
	Image        string // the compose reference as written
	Repo         string // normalized repo, e.g. library/redis (docker.io implied)
	LocalDigest  string // sha256:… or "" when unknown
	RemoteDigest string // sha256:… or "" when unknown
	State        ImageUpdateState
}

// ImageRef is a parsed compose image reference, with Docker's defaults
// applied: host docker.io, library/ prefix for official images, tag latest.
// A ref may carry both tag and digest.
type ImageRef struct {
	Host   string // "docker.io", "ghcr.io", "lscr.io", …
	Repo   string // "library/redis", "linuxserver/calibre", …
	Tag    string // "latest", "7-alpine", …
	Digest string // "sha256:…" or ""
}

// ParseImageRef applies Docker's reference grammar. The port-colon case
// (registry.example.com:5000/app:v1) is the reason this is a named, tested
// function and not inline string splitting — the tag separator is the LAST
// colon after the last slash, and a first-colon split turns
// registry.example.com:5000/app:v1 into a repo named registry.example and a
// tag 5000/app:v1 that every later comparison silently mismatches.
func ParseImageRef(image string) (ImageRef, error) {
	image = strings.TrimSpace(image)
	if image == "" {
		return ImageRef{}, fmt.Errorf("image reference is empty")
	}

	name, digest, hasDigest := strings.Cut(image, "@")
	if hasDigest {
		if err := checkDigest(digest); err != nil {
			return ImageRef{}, fmt.Errorf("image %q: %w", image, err)
		}
	}

	tag := ""
	if colon := strings.LastIndex(name, ":"); colon > strings.LastIndex(name, "/") {
		tag = name[colon+1:]
		name = name[:colon]
		if !validTag(tag) {
			return ImageRef{}, fmt.Errorf("image %q has an invalid tag %q", image, tag)
		}
	}

	host := ""
	if i := strings.Index(name, "/"); i >= 0 {
		candidate := name[:i]
		// Docker's own host rule: a leading component is a registry only
		// when it contains a dot or a colon, or is localhost. Without this
		// gate, prom/prometheus would resolve to a registry named prom.
		if strings.ContainsAny(candidate, ".:") || candidate == "localhost" {
			host = candidate
			name = name[i+1:]
		}
	}

	if name == "" {
		return ImageRef{}, fmt.Errorf("image %q has no repository", image)
	}

	if host == "" || host == "index.docker.io" {
		host = "docker.io"
	}
	// Docker's own default: a bare name on the Hub is an official image,
	// served as library/<name>. This is the repo part RepoDigests entries
	// are filtered against, and the Hub token scope is built from it.
	if host == "docker.io" && !strings.Contains(name, "/") {
		name = "library/" + name
	}
	if tag == "" {
		tag = "latest"
	}

	return ImageRef{Host: host, Repo: name, Tag: tag, Digest: digest}, nil
}

// IsPinnedToDigest reports whether ref carries a digest with no tag, which
// cannot drift and needs no network call. A tagged digest pin
// (redis:7-alpine@sha256:…) is deliberately NOT this: its tag can still move,
// and skipping it would make the feature one-shot for a service that was just
// updated (D4).
func IsPinnedToDigest(ref string) bool {
	name, digest, ok := strings.Cut(ref, "@")
	if !ok {
		return false
	}
	// An empty or malformed digest is not a pin; it is a ref the check will
	// classify Unknown anyway.
	if _, encoded, hasColon := strings.Cut(digest, ":"); !hasColon || encoded == "" {
		return false
	}

	return strings.LastIndex(name, ":") <= strings.LastIndex(name, "/")
}

// ClassifyImageUpdate is the whole decision. local/remote are "sha256:…" or
// ""; the empty string is a failed half — an image with no RepoDigests, or a
// registry that never answered — and the answer is Unknown, never Stale and
// never UpToDate (D3).
func ClassifyImageUpdate(local, remote string) ImageUpdateState {
	if local == "" || remote == "" {
		return ImageUnknown
	}

	if local == remote {
		return ImageUpToDate
	}

	return ImageStale
}

// localInspectTimeout bounds one `docker image inspect`. The command is
// milliseconds against a live daemon (R5); the timeout exists so a hung
// daemon cannot hang the startup check, and is generous for that reason.
const localInspectTimeout = 10 * time.Second

// LocalImageDigests resolves refs through `docker image inspect` and returns,
// per ref, the RepoDigests entry matching that repo — or "" when the image is
// absent locally or carries none (a locally built image, R4). Never indexes
// without a length check (R4).
//
// The inspects are per-reference rather than one batched call: docker writes
// its errors for missing refs to stderr and only what it found to stdout, so
// a batched call's stdout lines cannot be told apart once one ref is missing
// (R8). Each inspect is milliseconds; the local half never touches the
// network.
func LocalImageDigests(refs []string) (map[string]string, error) {
	digests := make(map[string]string, len(refs))
	seen := make(map[string]bool, len(refs))

	for _, ref := range refs {
		if seen[ref] {
			continue
		}
		seen[ref] = true

		// A ref that does not parse is "" without an exec: the registry
		// URL could not be built from it either, so both halves land in
		// the same Unknown.
		parsed, err := ParseImageRef(ref)
		if err != nil {
			digests[ref] = ""
			continue
		}

		digest, err := localImageDigest(ref, parsed.Repo)
		if err != nil {
			return nil, err
		}

		digests[ref] = digest
	}

	return digests, nil
}

// localImageDigest runs one inspect and picks repo's entry out of the
// RepoDigests it prints. A ref absent locally is "" and nil error — never
// pulled, or a build-only image — because the classification already has a
// name for that: Unknown. Any other failure aborts the batch: a daemon that
// is down fails every ref the same way, so the first one reports and the
// caller swallows it (D3) — DockerPreflight's existing surface says why,
// elsewhere.
func localImageDigest(ref string, repo string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), localInspectTimeout)
	defer cancel()

	command := dockerCommandContext(ctx, "docker", "image", "inspect", ref, "--format", "{{json .RepoDigests}}")
	output, err := command.CombinedOutput()
	if err != nil {
		// docker exits 1 for a missing ref too; CombinedOutput carries the
		// daemon's "No such image" line, which is how that case is told
		// apart from a daemon that is not running at all.
		if strings.Contains(strings.ToLower(string(output)), "no such image") {
			return "", nil
		}

		return "", fmt.Errorf("docker image inspect %s failed: %w: %s", ref, err, string(output))
	}

	return digestForRepo(string(output), repo), nil
}

// digestForRepo picks the RepoDigests entry whose repo part is the same
// image as repo, from one image's captured {{json .RepoDigests}} output.
//
// The entry's repo is normalized through ParseImageRef because the daemon
// writes the repo as it was pulled — "alpine@…" for library/alpine, but
// "ghcr.io/owner/app@…" with the host on — so neither side of the compare is
// usable raw (R4). Entries naming a different repo belong to a different
// pull and are skipped; the image this ref resolves to only carries entries
// for the repos it was actually pulled through, so the first match is the
// one. Malformed entries are skipped rather than fatal: the comparison
// degrades to Unknown, which is the contract for a failed half.
func digestForRepo(output string, repo string) string {
	var entries []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entries); err != nil {
		return ""
	}

	for _, entry := range entries {
		name, digest, ok := strings.Cut(entry, "@")
		if !ok {
			continue
		}

		parsed, err := ParseImageRef(name)
		if err != nil {
			continue
		}

		if parsed.Repo == repo {
			return digest
		}
	}

	return ""
}

// remoteWorkers caps the concurrent registry connections (D6). Serial by
// image is ~0.5 s each across a 24-service stack; unbounded invites Docker
// Hub's limiter. Four bounds a 20-service stack at ~3 s typical, ~25 s worst
// case.
const remoteWorkers = 4

// acceptedManifestTypes is the Accept header the manifest GET carries. Both
// index types are named because the digest that matches RepoDigests — the
// digest the registry serves for the tag — is the index digest (R2): asking
// for a single platform manifest would return that platform's digest and
// report every image stale forever.
const acceptedManifestTypes = "application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.index.v1+json"

// newRegistryClient is the seam that lets a test point the check at an
// httptest server, the way dockerCommand lets a test stand in for the docker
// binary (DockerCommand.go). The real client is plain net/http: the generic
// anonymous bearer flow (D4) is the whole client, and no credential of any
// kind is ever attached to it.
var newRegistryClient = func() *http.Client {
	return &http.Client{}
}

// ImageRefKey is the map key RemoteImageDigests files its answers under. Two
// refs that normalize alike (alpine:latest, alpine) are one registry call,
// and the key is what callers derive to read their ref's answer back.
func ImageRefKey(ref ImageRef) string {
	return ref.Host + "/" + ref.Repo + ":" + ref.Tag
}

// RemoteImageDigests returns the tag's current digest for each ref, using the
// generic anonymous bearer flow (D4). It has no error return: failures are
// per-ref and silent — a ref that failed carries "" and the caller's
// classification reads Unknown.
//
// Refs that normalize to the same host/repo:tag are answered by one registry
// call: two services on alpine:latest are one request, not two, and the
// check spends the minimum against a limiter whose budget is per IP.
func RemoteImageDigests(refs []ImageRef, timeout time.Duration) map[string]ImageUpdate {
	unique := make([]ImageRef, 0, len(refs))
	seen := make(map[string]bool, len(refs))
	for _, ref := range refs {
		key := ImageRefKey(ref)
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, ref)
	}

	// Results land in a slice indexed by position, not in a shared map, so
	// the workers never contend and the merge below is single-goroutine.
	digests := make([]string, len(unique))
	client := newRegistryClient()
	sem := make(chan struct{}, remoteWorkers)

	var wg sync.WaitGroup
	for i, ref := range unique {
		wg.Add(1)
		go func() {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			digests[i] = fetchRemoteDigest(client, ref, timeout)
		}()
	}
	wg.Wait()

	updates := make(map[string]ImageUpdate, len(unique))
	for i, ref := range unique {
		updates[ImageRefKey(ref)] = ImageUpdate{
			Repo:         ref.Repo,
			RemoteDigest: digests[i],
			State:        ImageUnknown,
		}
	}

	return updates
}

// registryAPIBase maps the ref's host onto the base URL its v2 API answers
// on. docker.io is a distribution name, not an API endpoint — the Hub's API
// lives on registry-1.docker.io (R1); every other registry serves /v2/ on
// the host it is named by, which is why there is no table of registries
// here and no per-registry branch anywhere in this file.
func registryAPIBase(host string) string {
	if host == "docker.io" {
		return "https://registry-1.docker.io"
	}

	return "https://" + host
}

// fetchRemoteDigest runs one ref's whole remote half: a manifest GET, and —
// when the registry answers 401 with a bearer challenge — the anonymous
// token fetch and the retry. There is no error return because there is no
// caller to give one to: any failure is "" and the classification reads
// Unknown (D3).
func fetchRemoteDigest(client *http.Client, ref ImageRef, timeout time.Duration) string {
	manifestURL := registryAPIBase(ref.Host) + "/v2/" + ref.Repo + "/manifests/" + ref.Tag

	digest, challenge, err := manifestRequest(client, manifestURL, timeout, "")
	if err == nil {
		return digest
	}
	if challenge == nil {
		return ""
	}

	token, err := bearerToken(client, challenge, timeout)
	if err != nil || token == "" {
		return ""
	}

	digest, _, err = manifestRequest(client, manifestURL, timeout, token)
	if err != nil {
		return ""
	}

	return digest
}

// manifestRequest GETs one manifest URL and reads its Docker-Content-Digest
// header. A 401 answered with a bearer challenge comes back as the
// challenge, for the caller to fetch a token and retry the same URL — the
// one generic flow R1 measured working unchanged against Docker Hub, ghcr.io
// and lscr.io.
func manifestRequest(client *http.Client, manifestURL string, timeout time.Duration, bearer string) (string, *authChallenge, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Accept", acceptedManifestTypes)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer func() {
		// The body is never read — the digest rides the header — but an
		// undrained body pins the connection instead of returning it.
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			return "", parseAuthChallenge(resp.Header.Get("WWW-Authenticate")), fmt.Errorf("registry %s answered %d", manifestURL, resp.StatusCode)
		}
		return "", nil, fmt.Errorf("registry answered %s: %d", manifestURL, resp.StatusCode)
	}

	digest := resp.Header.Get("Docker-Content-Digest")
	if digest == "" {
		return "", nil, fmt.Errorf("manifest %s carried no Docker-Content-Digest header", manifestURL)
	}

	return digest, nil, nil
}

// authChallenge is a WWW-Authenticate: Bearer challenge's useful parts.
type authChallenge struct {
	realm   string
	service string
	scope   string
}

// bearerToken fetches an anonymous token from the challenge's realm, passing
// the service and scope the registry itself named. A realm that is not HTTPS
// is refused: the flow was measured against HTTPS registries, and silently
// following a challenge to some other scheme is not a behavior this client
// should grow on its own.
func bearerToken(client *http.Client, challenge *authChallenge, timeout time.Duration) (string, error) {
	if !strings.HasPrefix(challenge.realm, "https://") {
		return "", fmt.Errorf("token realm %q is not HTTPS", challenge.realm)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	u, err := url.Parse(challenge.realm)
	if err != nil {
		return "", err
	}
	q := u.Query()
	if challenge.service != "" {
		q.Set("service", challenge.service)
	}
	if challenge.scope != "" {
		q.Set("scope", challenge.scope)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint answered %d", resp.StatusCode)
	}

	var token struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return "", err
	}
	if token.Token == "" && token.AccessToken == "" {
		return "", fmt.Errorf("token response carried no token")
	}
	if token.Token != "" {
		return token.Token, nil
	}

	return token.AccessToken, nil
}

// wwwAuthParam pulls key="value" pairs out of a WWW-Authenticate header.
var wwwAuthParam = regexp.MustCompile(`(\w+)="([^"]*)"`)

// parseAuthChallenge reads a Bearer challenge from a WWW-Authenticate
// header. Anything else — no header, a Basic challenge, no realm — returns
// nil: there is nothing to retry with, and the ref stays Unknown.
func parseAuthChallenge(header string) *authChallenge {
	if !strings.HasPrefix(strings.TrimSpace(header), "Bearer ") {
		return nil
	}

	challenge := &authChallenge{}
	for _, match := range wwwAuthParam.FindAllStringSubmatch(header, -1) {
		switch match[1] {
		case "realm":
			challenge.realm = match[2]
		case "service":
			challenge.service = match[2]
		case "scope":
			challenge.scope = match[2]
		}
	}

	if challenge.realm == "" {
		return nil
	}

	return challenge
}

// checkDigest validates the <alg>:<encoded> shape Docker's reference grammar
// gives digests. The algorithm part is lowercase alphanumerics joined by
// single separators; the encoded part is the grammar's [a-zA-Z0-9=_-]+. The
// sha256 hex length is not asserted here because the grammar does not: a
// digest that is the wrong length is still shaped like a digest, and the
// places that need a real one (the pin) get theirs from the registry's own
// Docker-Content-Digest header.
func checkDigest(digest string) error {
	alg, encoded, ok := strings.Cut(digest, ":")
	if !ok || alg == "" || encoded == "" {
		return fmt.Errorf("digest %q is not in <alg>:<encoded> form", digest)
	}

	for i, r := range alg {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		case (r == '.' || r == '_' || r == '-') && i > 0 && i < len(alg)-1:
		default:
			return fmt.Errorf("digest algorithm %q is not a valid algorithm name", alg)
		}
	}

	for _, r := range encoded {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '=', r == '_', r == '-':
		default:
			return fmt.Errorf("digest %q has an invalid encoded part", digest)
		}
	}

	return nil
}

// validTag is Docker's tag grammar: [a-zA-Z0-9_][a-zA-Z0-9._-]{0,127}.
func validTag(tag string) bool {
	if len(tag) == 0 || len(tag) > 128 {
		return false
	}

	for i, r := range tag {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
		case (r == '.' || r == '-') && i > 0:
		default:
			return false
		}
	}

	return true
}
