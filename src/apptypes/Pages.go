package apptypes

import "strings"

// PageTitles is the ordered list of page IDs used for navigation state.
var PageTitles = []string{
	"Home",
	"Dashboard",
	"Compose Files",
	"Settings",
}

// PageLabels maps page IDs to their display labels in the main menu.
// Page IDs are preserved for all state comparisons; only the label changes.
var PageLabels = map[string]string{
	"Home":          "Groups",
	"Dashboard":     "Dashboard",
	"Compose Files": "Files",
	"Settings":      "Settings",
}

// PageLabel returns a page's display label, falling back to the page ID.
func PageLabel(page string) string {
	if label, ok := PageLabels[page]; ok && label != "" {
		return label
	}

	return page
}

// PageShortcut returns the letter that jumps to a page: the first letter of its
// display label, lowercased. The nav underlines this letter in the label and
// AppModel binds it as alt+<letter>.
//
// It is derived from the label rather than listed in a table on purpose - a
// hand-maintained table drifts from the underline, and then the nav advertises
// a key that does nothing. TestPageShortcutsAreUnique guards the assumption
// that no two labels start with the same letter.
func PageShortcut(page string) string {
	label := PageLabel(page)
	if label == "" {
		return ""
	}

	return strings.ToLower(string([]rune(label)[0]))
}

// PageForShortcut returns the page a shortcut letter jumps to, or "" if the
// letter is not a page shortcut.
func PageForShortcut(letter string) string {
	for _, page := range PageTitles {
		if PageShortcut(page) == strings.ToLower(letter) {
			return page
		}
	}

	return ""
}
