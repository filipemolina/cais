package components

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestIndentAfter(t *testing.T) {
	const end = -100 // sentinel: use the rune length of the line

	tests := []struct {
		name string
		line string
		col  int
		want string
	}{
		{"leading spaces of the line, the common case", "  image: nginx", end, "  "},
		{"a top-level block opener deepens", "web:", end, "  "},
		{"a nested block opener deepens from its own indent", "  ports:", end, "    "},
		{"trailing spaces after the colon do not block the deepen", "  ports:   ", end, "    "},
		{"a trailing comment does not block the deepen", "  ports: # exposed", end, "    "},
		{"a comment never opens a block, even if it ends in a colon", "  # ports:", end, "  "},
		{"a sequence item aligns under its quoted content", `  - "8080:80"`, end, "    "},
		{"a sequence item ending in a value aligns under the key, no deepen", "  - name: web", end, "    "},
		{"a bare dash aligns to dash column plus one level", "  -", end, "    "},
		{"a whitespace-only line keeps its own spaces as the base", "    ", 4, "    "},
		{"an empty line has no indent", "", 0, ""},
		{"a mid-line split never deepens", "  image: nginx", 9, "  "},
		{"a mid-line split with no leading whitespace has an empty base", "web:", 2, ""},
		{"a column past the end of the line clamps to the end", "  image: nginx", 999, "  "},
		// The plan's table lists this row as clamping to the end-of-line
		// result ("  "), matching the col=999 row above. That contradicts
		// the plan's own stated rule ("col < 0 -> clamp to 0"): clamping to
		// 0 makes `before` empty, so the base is "". Negative columns can't
		// happen in practice (textarea.Column() never returns one); this
		// implementation follows the literal clamp-to-0 rule since it is
		// unambiguous, and documents the discrepancy here rather than
		// silently picking one.
		{"a negative column clamps to the start of the line", "  image: nginx", -1, ""},
		{"non-ASCII content does not shift the rune slice", "  # níveis:", end, "  "},

		// Resolving rule 2 vs rule 3 for a dash item whose own last token
		// opens a block: the base comes from rule 3 (the content column),
		// and rule 2 then deepens that base by one more level.
		{"a sequence item whose content opens a block aligns then deepens", "  - name:", end, "      "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col := tt.col
			if col == end {
				col = len([]rune(tt.line))
			}

			got := indentAfter(tt.line, col)
			if got != tt.want {
				t.Errorf("indentAfter(%q, %d) = %q, want %q", tt.line, col, got, tt.want)
			}
		})
	}
}

// TestIndentAfterBuildsValidYAML simulates typing a nested block by feeding
// each line's own end column back into indentAfter for the next line. It
// guards against a rule that is individually plausible but collectively
// produces invalid YAML.
func TestIndentAfterBuildsValidYAML(t *testing.T) {
	lines := []string{"services:"}
	additions := []string{
		"web:",
		"image: nginx",
		"ports:",
		`- "8080:80"`,
	}

	for _, next := range additions {
		last := lines[len(lines)-1]
		indent := indentAfter(last, len([]rune(last)))
		lines = append(lines, indent+next)
	}

	doc := strings.Join(lines, "\n") + "\n"

	var out map[string]any
	if err := yaml.Unmarshal([]byte(doc), &out); err != nil {
		t.Fatalf("built document is not valid YAML: %v\n%s", err, doc)
	}

	const want = "services:\n  web:\n    image: nginx\n    ports:\n      - \"8080:80\"\n"
	if doc != want {
		t.Fatalf("built document =\n%s\nwant\n%s", doc, want)
	}
}
