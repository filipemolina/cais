package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
