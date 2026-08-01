package chrome

import "github.com/charmbracelet/x/ansi"

// Hyperlink wraps already-sized text in an OSC 8 terminal hyperlink pointing
// at url. text must already be truncated and padded to its column - OSC 8's
// escape sequence is not text, and handing a not-yet-truncated string to
// runewidth.Truncate (chrome.Truncate) will cut through the middle of it,
// corrupting the rest of the screen. Terminals without OSC 8 support simply
// do not render the sequence, so text is what they show.
func Hyperlink(text, url string) string {
	return ansi.SetHyperlink(url) + text + ansi.ResetHyperlink()
}
