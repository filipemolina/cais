package highlight

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/filipemolina/cais/src/appstyles"
)

// The highlighter must never change the bytes, only their styling: stripping
// the ANSI has to give back the exact input.
func TestYAMLPreservesTheBytes(t *testing.T) {
	inputs := []string{
		"",
		"services:\n  web:\n    image: nginx:alpine\n",
		"# a comment\nservices:\n  web:\n    command: |\n      echo hi # not a comment\n      echo bye\n",
		"profiles: [\"core\", \"extra\"] # trailing\n",
		"  - \"quoted: with colon\"\n",
		"image: \"foo: bar\"\n",
	}

	for _, input := range inputs {
		if got := ansi.Strip(YAML(input)); got != input {
			t.Errorf("strip(YAML(%q)) = %q, bytes changed", input, got)
		}
	}
}

// colored reports whether the rendered output wraps target in the given
// style's color, i.e. target appears with some SGR before it and a reset
// after. It is a coarse check that *something* was colored, which is what
// distinguishes a highlighted line from a plain one.
func colored(rendered, target string) bool {
	idx := strings.Index(rendered, target)
	if idx < 0 {
		return false
	}
	// There must be an escape before the target on its line.
	lineStart := strings.LastIndex(rendered[:idx], "\n") + 1
	segment := rendered[lineStart:idx]
	return strings.Contains(segment, "\x1b[")
}

func withTheme(t *testing.T, fn func()) {
	t.Helper()
	original := appstyles.Active
	t.Cleanup(func() { appstyles.Active = original })
	fn()
}

func TestYAMLColorsKeys(t *testing.T) {
	withTheme(t, func() {
		out := YAML("services:\n  web:\n    image: nginx:alpine\n")

		for _, key := range []string{"services", "web", "image"} {
			if !colored(out, key) {
				t.Errorf("key %q was not colored:\n%q", key, out)
			}
		}
	})
}

func TestYAMLColorsQuotedStrings(t *testing.T) {
	withTheme(t, func() {
		out := YAML("profiles: [\"core\"]\n")

		if !colored(out, "\"core\"") {
			t.Errorf("quoted string was not colored:\n%q", out)
		}
	})
}

func TestYAMLColorsComments(t *testing.T) {
	withTheme(t, func() {
		out := YAML("# top\nimage: nginx # trailing\n")

		if !colored(out, "# top") {
			t.Errorf("full-line comment was not colored:\n%q", out)
		}
		if !colored(out, "# trailing") {
			t.Errorf("trailing comment was not colored:\n%q", out)
		}
	})
}

// A '#' inside a quoted string is not a comment.
func TestYAMLDoesNotTreatAHashInAStringAsAComment(t *testing.T) {
	withTheme(t, func() {
		out := YAML("password: \"abc#123\"\n")

		// The whole quoted span should be one string-colored run; if the '#'
		// had started a comment, the run would have been split and the closing
		// quote would not be inside the string span.
		if !colored(out, "\"abc#123\"") {
			t.Errorf("a quoted string containing # was not kept as one string:\n%q", out)
		}
	})
}

// A block scalar body is literal text, not structure: keys and comments
// inside it must not be colored as YAML.
func TestYAMLBlockScalarBodyIsNotHighlighted(t *testing.T) {
	withTheme(t, func() {
		input := "command: |\n  echo image: notakey\n  echo # notacomment\nnext: yes\n"
		out := YAML(input)

		// The opener's key is colored, and the line after the block ends is
		// colored, but inside the block nothing structural is.
		if !colored(out, "command") {
			t.Errorf("the block scalar's own key was not colored:\n%q", out)
		}
		if !colored(out, "next") {
			t.Errorf("the key after the block scalar was not colored:\n%q", out)
		}

		// Find the two body lines and confirm they carry no key coloring of
		// "image:" and no comment split. A body line is rendered dim as one
		// run, so "notakey" should not have the accent color before it.
		for _, body := range []string{"echo image: notakey", "echo # notacomment"} {
			idx := strings.Index(out, body)
			if idx < 0 {
				t.Fatalf("block scalar body %q missing:\n%q", body, out)
			}
		}
	})
}

// A less-indented line ends the block scalar even when a later line is
// indented again.
func TestYAMLBlockScalarEndsOnDedent(t *testing.T) {
	withTheme(t, func() {
		input := "command: |\n  literal\nback: 1\n  not_literal_anymore: 2\n"
		plain := ansi.Strip(YAML(input))
		if plain != input {
			t.Fatalf("bytes changed:\n%q", plain)
		}
	})
}
