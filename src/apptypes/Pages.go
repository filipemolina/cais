package apptypes

import "strings"

// Page is a page's identity - the value every screen-scoped switch matches on.
//
// It is a defined type rather than a bare string because five packages switch
// on it (AppModel routes keys and panels by it, cmds carries it, keys picks
// the footer's bindings from it, keybindingbar mirrors it, mainmenu renders
// the tabs from it) and every one of those switches falls through silently on
// a value it does not know. As a string that made a typo a page that quietly
// does nothing; as a type it is a compile error, and the constants below are
// the only spellings that exist.
//
// The underlying string is still the page's ID and is still what the display
// label falls back to, so an unlisted page renders as its own name rather than
// as blank.
type Page string

const (
	PageHome         Page = "Home"
	PageServices     Page = "Services"
	PageComposeFiles Page = "Compose Files"
	PageBackups      Page = "Backups"
)

// PageTitles is the ordered list of page IDs used for navigation state.
var PageTitles = []Page{
	PageHome,
	PageServices,
	PageComposeFiles,
	PageBackups,
}

// PageLabels maps page IDs to their display labels in the main menu.
// Page IDs are preserved for all state comparisons; only the label changes.
//
// The values are plain strings: a label is text on a tab, not an identity.
var PageLabels = map[Page]string{
	PageHome:         "Groups",
	PageServices:     "Services",
	PageComposeFiles: "Files",
}

// PageLabel returns a page's display label, falling back to the page ID.
func PageLabel(page Page) string {
	if label, ok := PageLabels[page]; ok && label != "" {
		return label
	}

	return string(page)
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
func PageShortcut(page Page) string {
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
func PageForShortcut(letter string) Page {
	for _, page := range PageTitles {
		if PageShortcut(page) == strings.ToLower(letter) {
			return page
		}
	}

	return ""
}
