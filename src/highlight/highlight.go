// Package highlight turns raw text into styled text for the Files page's
// read-only viewer. It is a display-layer concern only: the bytes are the
// file's own, the colors are the theme's.
//
// It is deliberately a hand-rolled, line-oriented highlighter rather than a
// real parser or a lexer library. A compose file is the app's one file to
// color, pulling in Chroma for one language is a heavy dependency against
// the project's minimal-deps stance (see TODO.md's dropped jq), and a
// best-effort line highlighter is honest about YAML's hard parts - it tracks
// block scalars so a `command: |` body is not colored as if it were
// structure, and degrades to plain text on anything it does not recognize
// rather than coloring it wrongly.
package highlight

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
)

// YAML returns content with syntax coloring applied as ANSI styling, using
// the active theme. The text itself is unchanged - every byte of the file is
// still there, in order - so the result is safe to scroll and to measure.
//
// Colors come from appstyles.Active read fresh on each call, so a theme
// switch repaints the file on the next content load.
func YAML(content string) string {
	if content == "" {
		return content
	}

	keyStyle := lipgloss.NewStyle().Foreground(appstyles.Active.Accent)
	strStyle := lipgloss.NewStyle().Foreground(appstyles.Active.StatusRunning)
	mutedStyle := lipgloss.NewStyle().Foreground(appstyles.Active.TextMuted)
	dimStyle := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim)

	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))

	// Block scalar state: after a `key: |` (or >, with chomping/indent
	// modifiers), the lines that belong to it are a literal string, not YAML.
	inBlock := false
	blockIndent := 0

	for _, line := range lines {
		indent := leadingSpaces(line)
		trimmed := strings.TrimSpace(line)

		if inBlock {
			// Blank lines and more-indented lines are part of the literal;
			// the first non-blank line at or below the opener's indent ends it.
			if trimmed == "" || indent > blockIndent {
				out = append(out, dimStyle.Render(line))
				continue
			}
			inBlock = false
		}

		switch {
		case trimmed == "":
			out = append(out, line)

		case strings.HasPrefix(trimmed, "#"):
			out = append(out, dimStyle.Render(line))

		default:
			code, comment := splitComment(line)
			styled := highlightCode(code, indent, keyStyle, strStyle, mutedStyle)
			if comment != "" {
				styled += dimStyle.Render(comment)
			}
			out = append(out, styled)

			if opensBlockScalar(code) {
				inBlock = true
				blockIndent = indent
			}
		}
	}

	return strings.Join(out, "\n")
}

// leadingSpaces counts the spaces a line is indented by. Tabs are counted as
// one each; YAML forbids them for indentation, so this only has to be good
// enough to compare block-scalar depth.
func leadingSpaces(line string) int {
	n := 0
	for n < len(line) && (line[n] == ' ' || line[n] == '\t') {
		n++
	}
	return n
}

// splitComment splits a line into the code before a comment and the comment
// itself (including its leading whitespace and the '#'). A '#' starts a
// comment when it is at the start or preceded by whitespace and is not
// inside a quoted string.
func splitComment(line string) (code, comment string) {
	var quote byte
	for i := 0; i < len(line); i++ {
		c := line[i]

		if quote != 0 {
			if c == '\\' && quote == '"' {
				i++ // skip the escaped char in a double-quoted string
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}

		switch c {
		case '"', '\'':
			quote = c
		case '#':
			if i == 0 || line[i-1] == ' ' || line[i-1] == '\t' {
				return line[:i], line[i:]
			}
		}
	}

	return line, ""
}

// opensBlockScalar reports whether a comment-stripped line ends with a block
// scalar indicator: `|` or `>`, optionally followed by chomping (+/-) and
// indentation (a digit) modifiers in either order.
func opensBlockScalar(code string) bool {
	token := lastToken(code)
	if token == "" || (token[0] != '|' && token[0] != '>') {
		return false
	}
	for _, r := range token[1:] {
		if r != '+' && r != '-' && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func lastToken(s string) string {
	s = strings.TrimRight(s, " \t")
	idx := strings.LastIndexAny(s, " \t")
	if idx < 0 {
		return s
	}
	return s[idx+1:]
}

// highlightCode styles the structural part of one line: the key of a
// key: value pair, a list marker, and the scalar that follows. indent is the
// line's leading-space count, re-applied unstyled so alignment is preserved.
func highlightCode(code string, indent int, keyStyle, strStyle, mutedStyle lipgloss.Style) string {
	lead := code[:indent]
	rest := code[indentedLen(code):]

	var b strings.Builder
	b.WriteString(lead)

	// A list item: "- thing". The marker is structural; the thing may itself
	// be a "key: value" (a mapping inside a list) or a scalar.
	if strings.HasPrefix(rest, "- ") || rest == "-" {
		b.WriteString(mutedStyle.Render("-"))
		if rest == "-" {
			return b.String()
		}
		b.WriteString(" ")
		rest = rest[2:]
	}

	// A mapping entry "key: value" (or "key:" with the value on later lines).
	// The colon that ends the key is the first one outside quotes that is
	// followed by a space or the end of the line.
	if keyEnd := mappingColon(rest); keyEnd >= 0 {
		b.WriteString(keyStyle.Render(rest[:keyEnd]))
		b.WriteString(mutedStyle.Render(":"))
		value := rest[keyEnd+1:]
		if value != "" {
			// value begins with the space after the colon.
			b.WriteString(highlightScalar(value, strStyle, mutedStyle))
		}
		return b.String()
	}

	// Not a mapping: a plain scalar or a continuation line.
	b.WriteString(highlightScalar(rest, strStyle, mutedStyle))
	return b.String()
}

// indentedLen is the byte length of the leading whitespace, matching what
// leadingSpaces counted.
func indentedLen(s string) int {
	n := 0
	for n < len(s) && (s[n] == ' ' || s[n] == '\t') {
		n++
	}
	return n
}

// mappingColon returns the index of the colon that ends a mapping key, or -1
// if rest is not a "key: ..." line. The key must be a single token (no
// unquoted spaces), which covers compose's service and field names.
func mappingColon(rest string) int {
	var quote byte
	for i := 0; i < len(rest); i++ {
		c := rest[i]

		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}

		switch c {
		case '"', '\'':
			quote = c
		case ' ', '\t':
			// An unquoted space before any colon means this is not a
			// "key: value" line (the key would be two tokens).
			return -1
		case ':':
			// The key's colon is followed by a space or ends the line.
			if i+1 >= len(rest) || rest[i+1] == ' ' {
				return i
			}
		}
	}
	return -1
}

// highlightScalar colors a scalar value: quoted strings in the string color,
// everything else muted. It walks the value so flow collections like
// ["a", "b"] get their quoted items colored while the brackets and bare
// words stay muted.
func highlightScalar(value string, strStyle, mutedStyle lipgloss.Style) string {
	var b strings.Builder
	var plain strings.Builder
	flushPlain := func() {
		if plain.Len() > 0 {
			b.WriteString(mutedStyle.Render(plain.String()))
			plain.Reset()
		}
	}

	for i := 0; i < len(value); i++ {
		c := value[i]
		if c != '"' && c != '\'' {
			plain.WriteByte(c)
			continue
		}

		// Found an opening quote: emit the muted run so far, then the quoted
		// span (through its closing quote) in the string color.
		flushPlain()
		quote := c
		j := i + 1
		for j < len(value) {
			if value[j] == '\\' && quote == '"' {
				j += 2
				continue
			}
			if value[j] == quote {
				break
			}
			j++
		}
		if j >= len(value) {
			j = len(value) - 1 // unterminated string: color to end of line
		}
		b.WriteString(strStyle.Render(value[i : j+1]))
		i = j
	}
	flushPlain()

	return b.String()
}
