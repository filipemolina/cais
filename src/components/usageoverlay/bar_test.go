package usageoverlay

import (
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
)

func TestBar(t *testing.T) {
	tests := []struct {
		used  int64
		total int64
		width int
		want  int // number of filled cells
	}{
		{50, 100, 10, 5},           // 50% = 5 filled
		{1, 1000, 10, 1},           // 1 byte = 1 filled (non-zero rule)
		{0, 1000, 10, 0},           // 0 = 0 filled
		{100, 100, 10, 10},         // 100% = 10 filled
		{50, 0, 10, 0},             // total=0 renders all shaded
		{150, 100, 10, 10},         // overflow is clamped to width
		{333, 1000, 10, 3},         // 33.3% rounds to 3 filled
		{667, 1000, 10, 7},         // 66.7% rounds to 7 filled
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := bar(tt.used, tt.total, tt.width)

			// Strip ANSI codes to count actual characters
			stripped := stripANSI(result)
			width := runewidth.StringWidth(stripped)

			if width != tt.width {
				t.Errorf("bar(%d, %d, %d) rendered width %d, want %d", tt.used, tt.total, tt.width, width, tt.width)
			}

			// Count filled cells (█ characters)
			filled := strings.Count(stripped, "█")
			if filled != tt.want {
				t.Errorf("bar(%d, %d, %d) got %d filled, want %d. Result: %q", tt.used, tt.total, tt.width, filled, tt.want, result)
			}
		})
	}
}

// stripANSI removes ANSI escape sequences from a string
func stripANSI(s string) string {
	// Simple ANSI escape removal: remove everything between \x1b[ and a letter
	var result strings.Builder
	var inEscape bool
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
		} else if inEscape {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}
