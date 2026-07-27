package keys

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"github.com/filipemolina/stack-stitcher/src/constants"
)

// entryIn finds binding's entry in scope, failing the test if it is absent.
func entryIn(t *testing.T, scope Scope, binding key.Binding) Entry {
	t.Helper()

	for _, entry := range scope.Entries {
		if sameBinding(entry.Binding, binding) {
			return entry
		}
	}

	t.Fatalf("scope %q has no entry for %q", scope.Title, binding.Help().Key)
	return Entry{}
}

func scopeTitled(t *testing.T, catalog []Scope, title string) Scope {
	t.Helper()

	for _, scope := range catalog {
		if scope.Title == title {
			return scope
		}
	}

	t.Fatalf("catalog has no %q scope", title)
	return Scope{}
}

// The catalog's promise: a key the user can press right now is marked
// available, and one that does nothing is dimmed. Pinned per scope so a
// regression names the row it broke.
func TestCatalogAvailability(t *testing.T) {
	t.Run("groups list with groups", func(t *testing.T) {
		catalog := Catalog(Context{Page: "Home", Focused: constants.COMPONENT_BODY_LIST})

		listScope := scopeTitled(t, catalog, "List")
		for _, binding := range []key.Binding{List.Select, List.New, List.Delete, List.Filter, List.Navigate, List.GoToStart, List.GoToEnd} {
			if !entryIn(t, listScope, binding).Available {
				t.Errorf("%q should be available on a populated groups list", binding.Help().Key)
			}
		}
		// No filter stands, so there is nothing to clear.
		if entryIn(t, listScope, List.ClearFilter).Available {
			t.Error("esc clear filter should be dimmed with no filter applied")
		}

		// The details keys need a subject, and the list has focus.
		details := scopeTitled(t, catalog, "Details")
		if entryIn(t, details, Details.Start).Available {
			t.Error("s start should be dimmed while the list has focus")
		}

		global := scopeTitled(t, catalog, "Global")
		if entryIn(t, global, Global.Back).Available {
			t.Error("esc back should be dimmed while the list has focus")
		}
		for _, binding := range []key.Binding{Global.NextPanel, Global.PrevPanel, Global.Quit, Global.ForceQuit, Global.Help} {
			if !entryIn(t, global, binding).Available {
				t.Errorf("%q should be available everywhere", binding.Help().Key)
			}
		}

		overlays := scopeTitled(t, catalog, "Overlays")
		if entryIn(t, overlays, Overlay.Submit).Available {
			t.Error("overlay keys should be dimmed on the main screen")
		}
	})

	t.Run("group details with a group selected", func(t *testing.T) {
		catalog := Catalog(Context{Page: "Home", Focused: constants.COMPONENT_BODY_DETAILS, Selected: true})

		details := scopeTitled(t, catalog, "Details")
		for _, binding := range []key.Binding{Details.Start, Details.Stop, Details.Restart, Details.Pull, Details.Remove, Details.Logs} {
			if !entryIn(t, details, binding).Available {
				t.Errorf("%q should be available with a group selected", binding.Help().Key)
			}
		}
		// EditService/EditFile only exist on the services panel.
		if entryIn(t, details, Details.EditService).Available {
			t.Error("e edit should be dimmed on the group panel")
		}

		global := scopeTitled(t, catalog, "Global")
		if !entryIn(t, global, Global.Back).Available {
			t.Error("esc back should be available on the details panel")
		}
	})

	t.Run("a filter stands on the list", func(t *testing.T) {
		catalog := Catalog(Context{Page: "Home", Focused: constants.COMPONENT_BODY_LIST, Filter: list.FilterApplied})

		listScope := scopeTitled(t, catalog, "List")
		if !entryIn(t, listScope, List.ClearFilter).Available {
			t.Error("esc clear filter should be available with a filter applied")
		}
		if entryIn(t, listScope, List.Filter).Available {
			t.Error("/ filter should be dimmed while a filter is applied")
		}
	})

	t.Run("an empty list suppresses what needs a row", func(t *testing.T) {
		catalog := Catalog(Context{Page: "Home", Focused: constants.COMPONENT_BODY_LIST, ListEmpty: true})

		listScope := scopeTitled(t, catalog, "List")
		if !entryIn(t, listScope, List.New).Available {
			t.Error("n new should be available on an empty list - it makes the first group")
		}
		for _, binding := range []key.Binding{List.Select, List.Delete, List.Filter} {
			if entryIn(t, listScope, binding).Available {
				t.Errorf("%q should be dimmed on an empty list", binding.Help().Key)
			}
		}
	})
}

// The alt chords are the aliases the footer has no room for; the overlay is
// where they are advertised, derived from the labels.
func TestCatalogListsTheChordAliases(t *testing.T) {
	catalog := Catalog(Context{Page: "Home", Focused: constants.COMPONENT_BODY_LIST})

	pages := scopeTitled(t, catalog, "Pages")

	var alias *Entry
	for i, entry := range pages.Entries {
		if strings.Contains(entry.Binding.Help().Desc, "alias") {
			alias = &pages.Entries[i]
		}
	}
	if alias == nil {
		t.Fatal("the Pages scope has no alias entry")
	}
	if !alias.Available {
		t.Error("the chord alias should never be dimmed")
	}

	for _, want := range []string{"alt+g", "alt+s", "alt+f"} {
		if !strings.Contains(alias.Binding.Help().Key, want[4:]) {
			t.Errorf("alias entry %q does not mention %q", alias.Binding.Help().Key, want)
		}
	}
}

// A binding identity is keystrokes plus help text: the overlay's dimming is
// only as good as this comparison.
func TestSameBinding(t *testing.T) {
	if !sameBinding(List.Select, List.Select) {
		t.Error("a binding should equal itself")
	}
	if sameBinding(List.Select, List.Delete) {
		t.Error("different bindings should not compare equal")
	}
	// ClearFilter and Cancel share esc but say different things.
	if sameBinding(List.ClearFilter, Overlay.Cancel) {
		t.Error("same key with different help is a different row")
	}
}
