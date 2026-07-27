package apptypes

import "testing"

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
