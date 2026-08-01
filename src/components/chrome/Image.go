package chrome

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// ShortImage renders an image reference to fit width columns, giving up
// whole parts of the reference in a fixed order rather than truncating the
// tail — the name is the part a reader needs and it is the part a plain
// truncation destroys first.
//
// The ladder, widest first, each rung tried in full - the first that fits
// wins:
//
//  0. as written, minus a ":latest" tag
//  1. drop the registry host
//  2. drop the namespace
//  3. drop the tag (or digest)
//  4. Truncate, as a backstop
func ShortImage(ref string, width int) string {
	if ref == "" {
		return ""
	}

	registry, namespace, name, suffix := splitRef(ref)

	rungs := []string{
		joinRef(registry, namespace, name, suffix),
		joinRef("", namespace, name, suffix),
		joinRef("", "", name, suffix),
		joinRef("", "", name, ""),
	}

	for _, candidate := range rungs {
		if runewidth.StringWidth(candidate) <= width {
			return candidate
		}
	}

	return Truncate(name, width)
}

func joinRef(registry, namespace, name, suffix string) string {
	parts := make([]string, 0, 3)
	if registry != "" {
		parts = append(parts, registry)
	}
	if namespace != "" {
		parts = append(parts, namespace)
	}
	parts = append(parts, name)

	return strings.Join(parts, "/") + suffix
}

// splitRef breaks an image reference into registry, namespace, name and a
// suffix (":tag" or "@" plus a seven-character digest stub - the same
// length git uses for a short hash), dropping a ":latest" tag outright: it
// is the tag docker assumes when none is given, so it costs columns and
// carries no information.
//
// "Registry" follows docker's own rule, not "the first path segment": a
// leading segment is a registry only when it has at least one sibling
// segment and contains a '.' or a ':', or is exactly "localhost" -
// otherwise a bare "name:tag" reference like "postgres:16-alpine" would
// have its tag mistaken for a registry port. The tag/digest split happens
// only on the last path segment, and only after the path is split on '/',
// so a registry port ("registry:5000/app") is never mistaken for a tag.
func splitRef(ref string) (registry, namespace, name, suffix string) {
	segments := strings.Split(ref, "/")

	if len(segments) > 1 {
		first := segments[0]
		if strings.ContainsAny(first, ".:") || first == "localhost" {
			registry = first
			segments = segments[1:]
		}
	}

	last := segments[len(segments)-1]
	namespace = strings.Join(segments[:len(segments)-1], "/")

	if idx := strings.Index(last, "@"); idx != -1 {
		name = last[:idx]
		digest := last[idx+1:]
		if di := strings.Index(digest, ":"); di != -1 {
			digest = digest[di+1:]
		}
		if len(digest) > 7 {
			digest = digest[:7]
		}
		suffix = "@" + digest
		return registry, namespace, name, suffix
	}

	if idx := strings.LastIndex(last, ":"); idx != -1 {
		name = last[:idx]
		if tag := last[idx+1:]; tag != "latest" {
			suffix = ":" + tag
		}
		return registry, namespace, name, suffix
	}

	name = last
	return registry, namespace, name, suffix
}
