package chrome

import (
	"image/color"

	"github.com/filipemolina/stack-stitcher/src/appstyles"
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
func Truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}

	return runewidth.Truncate(s, w, "…")
}
