package components

import "strings"

// yamlIndent is one level of YAML nesting. Two spaces is the compose
// convention and what the fragments the editor opens with already use, so it
// is a constant rather than a setting until someone asks for one.
const yamlIndent = "  "

// indentAfter returns the leading whitespace a new line should start with
// when Enter splits `line` at `col`. `col` is a rune index — the textarea's
// Column() is rune-based, not byte-based — and is clamped into range, since
// it comes from the textarea and should be treated as untrusted.
//
// The base indent comes from the text before the cursor (rule 1), a
// sequence item overrides that base to align under its own content (rule
// 3), and only then does a trailing ':' deepen one level (rule 2). Order
// matters: `- name: web` ends in a value, so it aligns under "name" without
// deepening, while `- name:` — a dash item whose own last token opens a
// block — aligns under "name" and then deepens once more, landing one level
// past it.
func indentAfter(line string, col int) string {
	runes := []rune(line)
	if col < 0 {
		col = 0
	}
	if col > len(runes) {
		col = len(runes)
	}

	before := string(runes[:col])

	// Rule 1: the base is the leading whitespace of the text before the
	// cursor. When `before` is entirely whitespace, that leading run is
	// `before` itself.
	base := before[:len(before)-len(strings.TrimLeft(before, " \t"))]

	// Rule 3: a sequence item's base is the column of its own content, not
	// just its leading whitespace, so a continuation line lands under the
	// item rather than under the dash.
	stripped := strings.TrimLeft(before, " \t")
	if stripped == "-" || strings.HasPrefix(stripped, "- ") {
		dashCol := len(base)
		rest := strings.TrimPrefix(stripped, "-")
		content := strings.TrimLeft(rest, " \t")

		var contentCol int
		if content == "" {
			// A bare dash, or a dash followed only by spaces, has no
			// content to align under: use the dash column plus one level,
			// not however many trailing spaces happen to follow it.
			contentCol = dashCol + 2
		} else {
			contentCol = dashCol + 1 + (len(rest) - len(content))
		}
		base = strings.Repeat(" ", contentCol)
	}

	// Rule 4: splitting a line mid-text never deepens, no matter what the
	// text before the cursor ends in — guessing a level deeper when the
	// user is breaking a line in half is worse than doing nothing.
	atEnd := strings.TrimLeft(string(runes[col:]), " \t") == ""
	if !atEnd {
		return base
	}

	// Rule 2: deepen one level if the meaningful text before the cursor
	// opens a block. "Meaningful" strips a trailing comment first — a plain
	// scan for '#', not a YAML tokenizer, so a quoted '#' in a value can
	// fool it. That is an accepted approximation (see yamlindent design
	// notes): the failure mode is one wrong indent level, fixed in one
	// shift+tab.
	meaningful := before
	if idx := strings.IndexRune(meaningful, '#'); idx >= 0 {
		meaningful = meaningful[:idx]
	}
	meaningful = strings.TrimRight(meaningful, " \t")

	if strings.HasSuffix(meaningful, ":") {
		base += yamlIndent
	}

	return base
}
