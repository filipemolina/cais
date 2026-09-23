package utils

import (
	"fmt"
	"strings"
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
