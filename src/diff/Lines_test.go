package diff

import (
	"strings"
	"testing"
)

// cases drives both the table test and the reconstruction property, so a
// case added for one is automatically covered by the other.
type linesCase struct {
	name   string
	before string
	after  string
	want   []Line
	// wantNil marks cases whose expected result is nil, which cannot be
	// spelled inside a []Line literal comparison.
	wantNil bool
}

var linesCases = []linesCase{
	{
		name:   "identical inputs are all Equal, in order, without newlines",
		before: "services:\n  web:\n    image: nginx\n",
		after:  "services:\n  web:\n    image: nginx\n",
		want: []Line{
			{Kind: Equal, Content: "services:"},
			{Kind: Equal, Content: "  web:"},
			{Kind: Equal, Content: "    image: nginx"},
		},
	},
	{
		name:   "a one-line change is Delete then Insert between Equals",
		before: "a: 1\nb: 2\nc: 3\n",
		after:  "a: 1\nb: 22\nc: 3\n",
		want: []Line{
			{Kind: Equal, Content: "a: 1"},
			{Kind: Delete, Content: "b: 2"},
			{Kind: Insert, Content: "b: 22"},
			{Kind: Equal, Content: "c: 3"},
		},
	},
	{
		name:   "a multi-line rewrite is one Delete block then one Insert block",
		before: "a\nb\nc\n",
		after:  "a\nx\ny\nc\n",
		want: []Line{
			{Kind: Equal, Content: "a"},
			{Kind: Delete, Content: "b"},
			{Kind: Insert, Content: "x"},
			{Kind: Insert, Content: "y"},
			{Kind: Equal, Content: "c"},
		},
	},
	{
		name:   "insert only",
		before: "a\nc\n",
		after:  "a\nb\nc\n",
		want: []Line{
			{Kind: Equal, Content: "a"},
			{Kind: Insert, Content: "b"},
			{Kind: Equal, Content: "c"},
		},
	},
	{
		name:   "delete only",
		before: "a\nb\nc\n",
		after:  "a\nc\n",
		want: []Line{
			{Kind: Equal, Content: "a"},
			{Kind: Delete, Content: "b"},
			{Kind: Equal, Content: "c"},
		},
	},
	{
		name:   "empty before is all Insert",
		before: "",
		after:  "x\ny\n",
		want: []Line{
			{Kind: Insert, Content: "x"},
			{Kind: Insert, Content: "y"},
		},
	},
	{
		name:   "empty after is all Delete",
		before: "x\ny\n",
		after:  "",
		want: []Line{
			{Kind: Delete, Content: "x"},
			{Kind: Delete, Content: "y"},
		},
	},
	{
		name:    "both empty is nil",
		before:  "",
		after:   "",
		wantNil: true,
	},
	{
		name:   "no trailing newline on either side",
		before: "a\nb",
		after:  "a\nb\nc",
		// The appended line lands at the boundary of a last line that has
		// no newline of its own, and udiff's line-boundary expansion covers
		// that edit by restating the line: a Delete+Insert pair with
		// identical content, then the added line. The reconstruction
		// property is what must hold here; the pairing heuristic that
		// smooths restated lines out is the rendering phase's deferred
		// intra-line work.
		want: []Line{
			{Kind: Equal, Content: "a"},
			{Kind: Delete, Content: "b"},
			{Kind: Insert, Content: "b"},
			{Kind: Insert, Content: "c"},
		},
	},
	{
		name:   "a whitespace-only change is still a change",
		before: "a: 1\nkey: value\n",
		after:  "a: 1\nkey: value \n",
		want: []Line{
			{Kind: Equal, Content: "a: 1"},
			{Kind: Delete, Content: "key: value"},
			{Kind: Insert, Content: "key: value "},
		},
	},
	{
		name:   "blank lines take part in the diff like any other line",
		before: "a\n\nb\n",
		after:  "a\nb\n",
		want: []Line{
			{Kind: Equal, Content: "a"},
			{Kind: Delete, Content: ""},
			{Kind: Equal, Content: "b"},
		},
	},
}

func TestLines(t *testing.T) {
	for _, tc := range linesCases {
		t.Run(tc.name, func(t *testing.T) {
			got := Lines(tc.before, tc.after)

			if tc.wantNil {
				if got != nil {
					t.Fatalf("Lines(%q, %q) = %v, want nil", tc.before, tc.after, got)
				}
				return
			}

			if len(got) != len(tc.want) {
				t.Fatalf("Lines(%q, %q) returned %d lines, want %d: %+v",
					tc.before, tc.after, len(got), len(tc.want), got)
			}
			for i, want := range tc.want {
				if !lineEqual(got[i], want) {
					t.Errorf("line %d = %+v, want %+v", i, got[i], want)
				}
			}
		})
	}
}

// lineEqual compares field by field: Line carries a slice, so it cannot be
// compared with !=.
func lineEqual(got, want Line) bool {
	if got.Kind != want.Kind || got.Content != want.Content {
		return false
	}
	return len(got.Spans) == 0 && len(want.Spans) == 0
}

// TestIdenticalInputsHaveNoChanges pins the cheap win built on this package:
// hashing the live file recognises "this copy is the file I have now", so an
// identical pair must not smuggle in any Delete or Insert line.
func TestIdenticalInputsHaveNoChanges(t *testing.T) {
	got := Lines("services:\n  web: {}\n", "services:\n  web: {}\n")
	for _, line := range got {
		if line.Kind != Equal {
			t.Errorf("identical inputs produced a %v line: %+v", line.Kind, line)
		}
	}
}

// TestLinesReconstructBothSides is the property that makes a whole-file
// render correct: the lines a renderer would draw for each side - Equal plus
// that side's changes, joined back with the newlines Content does not carry -
// rebuild each input minus its final newline. If this fails, the engine
// dropped or duplicated content and the rendering is wrong, not the test.
func TestLinesReconstructBothSides(t *testing.T) {
	for _, tc := range linesCases {
		t.Run(tc.name, func(t *testing.T) {
			got := Lines(tc.before, tc.after)

			if before := joinKinds(got, Equal, Delete); before != strings.TrimSuffix(tc.before, "\n") {
				t.Errorf("Equal+Delete reconstructed %q, want %q", before, strings.TrimSuffix(tc.before, "\n"))
			}
			if after := joinKinds(got, Equal, Insert); after != strings.TrimSuffix(tc.after, "\n") {
				t.Errorf("Equal+Insert reconstructed %q, want %q", after, strings.TrimSuffix(tc.after, "\n"))
			}
		})
	}
}

// joinKinds keeps the lines whose Kind matches and joins their Content, the
// way a per-line renderer would draw them back into a file.
func joinKinds(lines []Line, kinds ...Kind) string {
	var parts []string
	for _, line := range lines {
		for _, kind := range kinds {
			if line.Kind == kind {
				parts = append(parts, line.Content)
				break
			}
		}
	}
	return strings.Join(parts, "\n")
}
