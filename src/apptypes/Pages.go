package apptypes

import "strings"

// PageTitles is the ordered list of page IDs used for navigation state.
var PageTitles = []string{
	"Home",
	"Services",
	"Compose Files",
	"Env",
}

// PageLabels maps page IDs to their display labels in the main menu.
// Page IDs are preserved for all state comparisons; only the label changes.
var PageLabels = map[string]string{
	"Home":          "Groups",
	"Services":      "Services",
	"Compose Files": "Files",
}

// PageLabel returns a page's display label, falling back to the page ID.
func PageLabel(page string) string {
	if label, ok := PageLabels[page]; ok && label != "" {
		return label
	}

	return page
}

// PageShortcut returns the letter of a page's alt+<letter> chord: the first
// letter of its display label, lowercased.
//
// The chord is an alias. The digits are the primary page scheme, rendered on
// the tabs themselves; the chord stays for the terminals that send Option as
// Alt. It is still derived from the label rather than listed in a table,
// because a hand-maintained table drifts. Two labels may now share a first
// letter - the digits are unambiguous, and the alias simply resolves to the
// first matching page.
func PageShortcut(page string) string {
	label := PageLabel(page)
	if label == "" {
		return ""
	}

	return strings.ToLower(string([]rune(label)[0]))
}

// PageForShortcut returns the page a chord letter jumps to, or "" if the
// letter is not a page chord. When two labels share a first letter the first
// page wins - acceptable for an alias, since the digits are the primary
// scheme and are unambiguous.
func PageForShortcut(letter string) string {
	for _, page := range PageTitles {
		if PageShortcut(page) == strings.ToLower(letter) {
			return page
		}
	}

	return ""
}
