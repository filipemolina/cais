package chrome

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/filipemolina/cais/src/appstyles"
)

// updateGlyph is the character the update-available glyph draws with: an
// arrow pointing at the new version, not at the service. It is deliberately
// NOT a recoloured state dot - state colors are a fixed vocabulary (green
// runs, red stops; the dead-vs-stopped distinction survives only because the
// fault carries its own glyph), and a second meaning on one dot would re-open
// an argument DESIGN.md has settled.
const updateGlyph = "↑"

// UpdateGlyph renders the update-available glyph on bg, in the amber the
// status vocabulary already uses for "wants attention, not broken" - the same
// register the transitional states ride. It renders nothing for any state
// that is not a confirmed update: an Unknown answer is the network saying it
// does not know, and a wall of false glyphs would kill trust in the feature
// faster than any missed one.
func UpdateGlyph(bg color.Color) string {
	return lipgloss.NewStyle().
		Foreground(appstyles.Active.StatusStarting).
		Background(bg).
		Render(updateGlyph)
}

// ShortDigest renders a "sha256:…" digest at the twelve hex characters the
// member table's updates table shows. The full value lives in the service
// details panel; twelve characters are what a row can afford and what two
// digests can be told apart by.
func ShortDigest(digest string) string {
	digest = strings.TrimPrefix(digest, "sha256:")
	if len(digest) > 12 {
		digest = digest[:12]
	}

	return digest
}
