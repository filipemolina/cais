// Package diff turns two versions of a file into the whole-file line
// sequence the Backups preview renders: every line of the file in order,
// tagged with what turning one version into the other does to it.
//
// It is deliberately UI-free - no theme, no lipgloss, no bubbles. The
// rendering phase (docs/plans/backups-rework.md, Phase 5) colours these
// lines; the engine stays pure and cheap to test on its own.
package diff

import (
	"strings"

	udiff "github.com/aymanbagabas/go-udiff"
)

// Kind classifies one line of a whole-file diff.
type Kind int

const (
	// Equal is a line both sides of the diff agree on.
	Equal Kind = iota
	// Insert is a line the second side has that the first does not. In the
	// preview's direction (live, backup) it is content a restore would add.
	Insert
	// Delete is a line the first side has that the second does not. In the
	// preview's direction it is content a restore would remove from the
	// live file.
	Delete
)

// Span is a half-open byte range within a Line's Content that carries an
// intra-line change.
//
// Spans is present and empty from day one so the intra-line highlighting
// work (docs/plans/backups-rework.md, *Deferred*) fills a field rather than
// changing this type's shape - and so every caller that ignores it today is
// already correct when it starts arriving.
type Span struct {
	Start int
	End   int
}

// Line is one line of the whole-file view of a diff.
type Line struct {
	Kind Kind
	// Content is the line's text without its trailing newline: the renderer
	// draws one Line per row, and a newline inside Content would render as
	// a blank row.
	Content string
	Spans   []Span
}

// Lines diffs before against after into the sequence of lines a whole-file
// view renders: every line in file order, each tagged Equal, Insert or
// Delete.
//
// Called as Lines(live, backup) it answers "what does restoring this copy
// do to my file right now?" - a Delete line is content the restore would
// remove from the live file, an Insert line is content it would add, and an
// Equal line is content both agree on.
//
// The result covers the whole file, not hunks: the panel renders every line
// and scrolls to the first change, so Equal context is kept without a
// context limit.
//
// Chunking follows udiff's line-boundary expansion: an edit inside a line
// restates that line as a Delete+Insert pair with identical content. The
// case that reaches real files is an append to a file whose last line has no
// newline (see Lines_test.go). Smoothing those pairs out is a pairing
// heuristic, and it belongs to the rendering phase's deferred intra-line
// work, not to this engine.
//
// Lines returns nil when there is nothing to diff (both inputs empty), and
// on the unreachable internal error below. Callers read nil as "diff
// unavailable" and fall back to showing the copy's plain bytes - the same
// degradation a missing live file gets - rather than as a separate error.
func Lines(before, after string) []Line {
	edits := udiff.Lines(before, after)
	if len(edits) == 0 {
		// No edits means the two inputs are the same text; ToUnifiedDiff
		// would return no hunks at all, and the whole-file contract needs
		// the file's lines back as Equal so the caller can still render it.
		return equalLines(before)
	}

	// A context count at least the file's own line count makes every hunk
	// merge into one that spans the whole file, so the Equal context
	// reconstructs both sides end to end and the caller sees one sequence
	// of lines instead of hunks to stitch.
	contextLines := strings.Count(before, "\n") + 1
	ud, err := udiff.ToUnifiedDiff("", "", before, edits, contextLines)
	if err != nil {
		// Unreachable: the edits come from udiff.Lines itself, so they are
		// consistent by construction. nil is the callers' "diff
		// unavailable" signal (see above), not a crash path.
		return nil
	}

	var out []Line
	for _, hunk := range ud.Hunks {
		for _, line := range hunk.Lines {
			out = append(out, Line{
				Kind:    kindFor(line.Kind),
				Content: strings.TrimSuffix(line.Content, "\n"),
				Spans:   nil,
			})
		}
	}
	return out
}

// kindFor maps udiff's operation kinds onto this package's. The switch is
// the mapping: udiff numbers them Delete=0, Insert=1, Equal=2, and leaning
// on numeric identity across two packages would be invisible if either
// reordered.
func kindFor(op udiff.OpKind) Kind {
	switch op {
	case udiff.Insert:
		return Insert
	case udiff.Equal:
		return Equal
	default:
		return Delete
	}
}

// equalLines splits text the way udiff's line splitter does - on newlines,
// the final newline not producing a phantom empty line - and tags every
// line Equal.
func equalLines(text string) []Line {
	if text == "" {
		return nil
	}

	lines := strings.Split(text, "\n")
	if text[len(text)-1] == '\n' {
		// "a\nb\n" splits to ["a", "b", ""]: the trailing empty element is
		// the artifact of the final newline, not a blank line.
		lines = lines[:len(lines)-1]
	}

	out := make([]Line, len(lines))
	for i, content := range lines {
		out[i] = Line{Kind: Equal, Content: content}
	}
	return out
}
