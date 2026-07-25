package apptypes

import "testing"

// PageShortcut derives the alt+<letter> chord from the first letter of a page's
// label, so two labels starting with the same letter would make one page
// unreachable while the nav still underlined the letter on both. Renaming a
// label is exactly when that happens, so it is asserted rather than assumed.
func TestPageShortcutsAreUnique(t *testing.T) {
	seen := map[string]string{}

	for _, page := range PageTitles {
		key := PageShortcut(page)

		if key == "" {
			t.Errorf("page %q has no shortcut letter", page)
			continue
		}

		if other, clash := seen[key]; clash {
			t.Errorf("pages %q and %q both use alt+%s; give one a label starting with a different letter", other, page, key)
		}

		seen[key] = page
	}
}

func TestPageForShortcutRoundTrips(t *testing.T) {
	for _, page := range PageTitles {
		if got := PageForShortcut(PageShortcut(page)); got != page {
			t.Errorf("PageForShortcut(%q) = %q, want %q", PageShortcut(page), got, page)
		}
	}

	if got := PageForShortcut("z"); got != "" {
		t.Errorf("unbound letter should return no page, got %q", got)
	}
}

func TestPageLabelFallsBackToID(t *testing.T) {
	if got, want := PageLabel("Home"), "Groups"; got != want {
		t.Errorf("PageLabel(Home) = %q, want %q", got, want)
	}

	if got, want := PageLabel("Unlisted"), "Unlisted"; got != want {
		t.Errorf("PageLabel of an unlisted page = %q, want %q", got, want)
	}
}
