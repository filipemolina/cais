package utils

import (
	"bytes"
	"reflect"

	"gopkg.in/yaml.v3"
)

// blankLineMarker stands in for a blank line while a compose file is held as
// a yaml.Node.
//
// yaml.v3 round-trips comments but not blank lines, so re-encoding a
// document collapses the spacing the user put between their services. Since
// comments do survive, turning each blank line into a comment on the way in
// and back into a blank line on the way out preserves the spacing through an
// edit. The marker is deliberately unlovely so it cannot plausibly collide
// with a comment somebody actually wrote.
const blankLineMarker = "#__stack_stitcher_blank_line__"

// markBlankLines replaces blank lines with the marker comment, and reports
// whether doing so was safe.
//
// It is not always safe: a blank line inside a block scalar
//
//	command: |
//	  echo one
//
//	  echo two
//
// is part of the string, and commenting it out would silently rewrite the
// user's data. Rather than trying to track block scalars by hand - which
// means re-implementing a chunk of the YAML grammar, and being wrong about
// it corrupts files - the marked document is parsed and compared with the
// original. If the values differ, marking touched something it shouldn't
// have and the caller keeps the original text, losing the blank lines but
// nothing else.
func markBlankLines(raw []byte) ([]byte, bool) {
	lines := bytes.Split(raw, []byte("\n"))
	marked := make([][]byte, len(lines))
	found := false

	for i, line := range lines {
		// The final element after a trailing newline is an empty string that
		// is not a line at all; marking it would append a stray comment.
		if len(bytes.TrimSpace(line)) == 0 && i != len(lines)-1 {
			marked[i] = []byte(blankLineMarker)
			found = true
			continue
		}

		marked[i] = line
	}

	if !found {
		return raw, false
	}

	candidate := bytes.Join(marked, []byte("\n"))
	if !sameYAMLValues(raw, candidate) {
		return raw, false
	}

	return candidate, true
}

// unmarkBlankLines turns the marker comments back into blank lines. Safe to
// call on output that never had any.
func unmarkBlankLines(encoded []byte) []byte {
	if !bytes.Contains(encoded, []byte(blankLineMarker)) {
		return encoded
	}

	lines := bytes.Split(encoded, []byte("\n"))
	for i, line := range lines {
		if bytes.Equal(bytes.TrimSpace(line), []byte(blankLineMarker)) {
			lines[i] = nil
		}
	}

	return bytes.Join(lines, []byte("\n"))
}

// sameYAMLValues reports whether two documents carry the same data, ignoring
// comments and formatting.
func sameYAMLValues(a, b []byte) bool {
	var valueA, valueB any

	if err := yaml.Unmarshal(a, &valueA); err != nil {
		return false
	}
	if err := yaml.Unmarshal(b, &valueB); err != nil {
		return false
	}

	return reflect.DeepEqual(valueA, valueB)
}
