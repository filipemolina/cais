package chrome

import (
	"image/color"

	"github.com/charmbracelet/x/ansi"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/mattn/go-runewidth"
)

func HealthColor(health string) color.Color {
	switch health {
	case "healthy":
		return appstyles.Active.StatusRunning
	case "unhealthy":
		return appstyles.Active.StatusError
	case "starting":
		return appstyles.Active.StatusStarting
	default:
		return appstyles.Active.TextDim
	}
}

// Truncate hard-truncates s to w display columns, appending an ellipsis
// when it is shortened. lipgloss Width wraps rather than truncates, so
// cells are pre-truncated to keep every row on a single line.
//
// The ansi.StringWidth check is a fast path, not an alternate algorithm: a
// plain string that already fits returns unchanged either way, but a
// string carrying a zero-width escape sequence - an OSC 8 hyperlink built
// by chrome.Hyperlink, already sized to its column before being wrapped -
// does not, because runewidth.Truncate is not ANSI-aware and would cut
// through the middle of the sequence. See docs/plans/service-urls.md D5,
// which is why Hyperlink's own doc comment requires already-sized text
// rather than truncating after wrapping.
func Truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if ansi.StringWidth(s) <= w {
		return s
	}

	return runewidth.Truncate(s, w, "…")
}
