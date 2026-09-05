package keys

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"github.com/filipemolina/cais/src/apptypes"
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
		catalog := Catalog(Context{Page: "Home"})

		groups := scopeTitled(t, catalog, "Groups")
		for _, binding := range []key.Binding{List.New, List.Filter, List.Navigate, List.GoToStart, List.GoToEnd} {
			if !entryIn(t, groups, binding).Available {
				t.Errorf("%q should be available on a populated groups list", binding.Help().Key)
			}
		}
		// No filter stands, so there is nothing to clear.
		if entryIn(t, groups, List.ClearFilter).Available {
			t.Error("esc clear filter should be dimmed with no filter applied")
		}
		// Edit/Delete act on the row under the cursor, and a populated list
		// always has one - there is no way to deselect. The dimmed case is an
		// empty list, covered in its own subtest below.
		for _, binding := range []key.Binding{List.Edit, List.Delete} {
			if !entryIn(t, groups, binding).Available {
				t.Errorf("%q should be available on a populated list", binding.Help().Key)
			}
		}

		// Same for the action keys: the subject is whatever the cursor is on.
		if !entryIn(t, groups, Details.Start).Available {
			t.Error("s start should be available on a populated list")
		}
		// The same binding on another page's scope is dimmed because that
		// page is not the one on screen.
		services := scopeTitled(t, catalog, "Services")
		if entryIn(t, services, Details.Start).Available {
			t.Error("s start should be dimmed in the Services scope while Groups is up")
		}

		global := scopeTitled(t, catalog, "Global")
		// Back is only live with a selection or an applied filter.
		if entryIn(t, global, Global.Back).Available {
			t.Error("esc back should be dimmed with no selection and no filter")
		}
		for _, binding := range []key.Binding{Global.Quit, Global.ForceQuit, Global.Help, Global.About, Global.Theme} {
			if !entryIn(t, global, binding).Available {
				t.Errorf("%q should be available everywhere", binding.Help().Key)
			}
		}
		// Tab is dead on body pages now.
		for _, binding := range []key.Binding{Global.NextPanel, Global.PrevPanel} {
			if entryIn(t, global, binding).Available {
				t.Errorf("%q should be dimmed on body pages", binding.Help().Key)
			}
		}

		overlays := scopeTitled(t, catalog, "Overlays")
		if entryIn(t, overlays, Overlay.Submit).Available {
			t.Error("overlay keys should be dimmed on the main screen")
		}
	})

	t.Run("group details with a group selected", func(t *testing.T) {
		catalog := Catalog(Context{Page: "Home"})

		groups := scopeTitled(t, catalog, "Groups")
		for _, binding := range []key.Binding{Details.Start, Details.Stop, Details.Restart, Details.Pull, Details.Remove, Details.Logs} {
			if !entryIn(t, groups, binding).Available {
				t.Errorf("%q should be available with a group selected", binding.Help().Key)
			}
		}
		for _, binding := range []key.Binding{List.Edit, List.New, List.Delete} {
			if !entryIn(t, groups, binding).Available {
				t.Errorf("%q should be available with a group selected", binding.Help().Key)
			}
		}

		// Healthcheck, Boot, EditFile and CopyURL are service-only, so they
		// live in the Services scope - dimmed here because the Groups page
		// is the one on screen.
		services := scopeTitled(t, catalog, "Services")
		for _, binding := range []key.Binding{Details.Healthcheck, Details.Boot, Details.EditFile, Details.CopyURL} {
			if entryIn(t, services, binding).Available {
				t.Errorf("service-only verb %q should be dimmed while Groups is up", binding.Help().Key)
			}
		}

		// esc is not offered on a body page any more: there is no deselect
		// rung for it, and an applied filter takes the slot as "esc clear
		// filter" when there is one.
		global := scopeTitled(t, catalog, "Global")
		if entryIn(t, global, Global.Back).Available {
			t.Error("esc back should be dimmed on a body page")
		}
	})

	t.Run("a filter stands on the list", func(t *testing.T) {
		catalog := Catalog(Context{Page: "Home", Filter: list.FilterApplied})

		groups := scopeTitled(t, catalog, "Groups")
		if !entryIn(t, groups, List.ClearFilter).Available {
			t.Error("esc clear filter should be available with a filter applied")
		}
		if entryIn(t, groups, List.Filter).Available {
			t.Error("/ filter should be dimmed while a filter is applied")
		}
		// Back is claimed by the filter's clear slot, not the deselect slot.
		global := scopeTitled(t, catalog, "Global")
		if entryIn(t, global, Global.Back).Available {
			t.Error("esc back should be dimmed while a filter holds the esc slot")
		}
	})

	t.Run("an empty list suppresses what needs a row", func(t *testing.T) {
		catalog := Catalog(Context{Page: "Home", ListEmpty: true})

		groups := scopeTitled(t, catalog, "Groups")
		if !entryIn(t, groups, List.New).Available {
			t.Error("n new should be available on an empty list - it makes the first group")
		}
		for _, binding := range []key.Binding{List.Edit, List.Delete, List.Filter} {
			if entryIn(t, groups, binding).Available {
				t.Errorf("%q should be dimmed on an empty list", binding.Help().Key)
			}
		}
	})

	t.Run("service details while inline editing dims Global panel keys", func(t *testing.T) {
		catalog := Catalog(Context{Page: "Services", Editing: true})

		global := scopeTitled(t, catalog, "Global")
		for _, binding := range []key.Binding{Global.NextPanel, Global.PrevPanel} {
			if entryIn(t, global, binding).Available {
				t.Errorf("%q should be dimmed while the editor owns the keyboard", binding.Help().Key)
			}
		}
		// Back is still available: esc cancels the editor.
		if !entryIn(t, global, Global.Back).Available {
			t.Error("esc back should be available while editing")
		}

		// Editor scope entries are lit.
		editor := scopeTitled(t, catalog, "Editor")
		for _, binding := range []key.Binding{Editor.Indent, Editor.Outdent} {
			if !entryIn(t, editor, binding).Available {
				t.Errorf("%q should be available while editing", binding.Help().Key)
			}
		}
		// Save and OpenEditor are shared with Details; they should also be lit.
		for _, binding := range []key.Binding{Details.Save, Details.OpenEditor} {
			if !entryIn(t, editor, binding).Available {
				t.Errorf("%q should be available while editing", binding.Help().Key)
			}
		}
		// The docker verbs are not live while editing, even on their own page.
		services := scopeTitled(t, catalog, "Services")
		if entryIn(t, services, Details.Start).Available {
			t.Error("s start should be dimmed while editing")
		}
	})

	// The page scopes are the whole grouping now: a row is lit only while its
	// own page is up, so the same binding on a sibling page's scope is dimmed.
	t.Run("the services page lights its own scope only", func(t *testing.T) {
		catalog := Catalog(Context{Page: "Services"})

		services := scopeTitled(t, catalog, "Services")
		if !entryIn(t, services, Details.Start).Available {
			t.Error("s start should be available with a service selected")
		}

		groups := scopeTitled(t, catalog, "Groups")
		if entryIn(t, groups, Details.Start).Available {
			t.Error("s start should be dimmed in the Groups scope while Services is up")
		}

		files := scopeTitled(t, catalog, "Files")
		for _, binding := range []key.Binding{Details.EditFile, Files.Browse, Files.Scroll} {
			if entryIn(t, files, binding).Available {
				t.Errorf("%q should be dimmed while Services is up", binding.Help().Key)
			}
		}

		backups := scopeTitled(t, catalog, "Backups")
		for _, binding := range []key.Binding{Backup.Restore, Backup.Navigate} {
			if entryIn(t, backups, binding).Available {
				t.Errorf("%q should be dimmed while Services is up", binding.Help().Key)
			}
		}
	})

	t.Run("the files page lights its own scope only", func(t *testing.T) {
		catalog := Catalog(Context{Page: "Compose Files"})

		files := scopeTitled(t, catalog, "Files")
		for _, binding := range []key.Binding{Details.EditFile, Files.Browse, Files.Scroll} {
			if !entryIn(t, files, binding).Available {
				t.Errorf("%q should be available on the Files page", binding.Help().Key)
			}
		}

		// The Files page has no list, and it is not the Groups page either.
		groups := scopeTitled(t, catalog, "Groups")
		for _, binding := range []key.Binding{List.New, List.Navigate, Details.Start} {
			if entryIn(t, groups, binding).Available {
				t.Errorf("%q should be dimmed while Files is up", binding.Help().Key)
			}
		}
	})

	t.Run("the backups page lights its own scope only", func(t *testing.T) {
		catalog := Catalog(Context{Page: "Backups"})

		backups := scopeTitled(t, catalog, "Backups")
		for _, binding := range []key.Binding{Backup.Restore, Backup.Navigate} {
			if !entryIn(t, backups, binding).Available {
				t.Errorf("%q should be available on the Backups page", binding.Help().Key)
			}
		}

		groups := scopeTitled(t, catalog, "Groups")
		for _, binding := range []key.Binding{List.New, List.Navigate, Details.Start} {
			if entryIn(t, groups, binding).Available {
				t.Errorf("%q should be dimmed while Backups is up", binding.Help().Key)
			}
		}
	})
}

// TestEditorKeysAreLiveOnlyWhileEditing asserts the Editor scope's entries are
// available while editing and dimmed otherwise.
func TestEditorKeysAreLiveOnlyWhileEditing(t *testing.T) {
	editingCtx := Context{Page: "Services", Editing: true}
	catalog := Catalog(editingCtx)

	editor := scopeTitled(t, catalog, "Editor")
	for _, binding := range []key.Binding{Editor.Indent, Editor.Outdent} {
		if !entryIn(t, editor, binding).Available {
			t.Errorf("%q should be available while editing", binding.Help().Key)
		}
	}
	// Save and OpenEditor are shared with Details; they should also be lit.
	for _, binding := range []key.Binding{Details.Save, Details.OpenEditor} {
		if !entryIn(t, editor, binding).Available {
			t.Errorf("%q should be available while editing", binding.Help().Key)
		}
	}
	// Editor.NewLine (enter) is handled by the textarea internally, not through
	// the key binding system, so it is not in pressableNow. It appears in the
	// overlay but dimmed, which is fine: the overlay says it exists without
	// making a separate availability claim.

	// Without editing, all Editor scope entries are dimmed.
	notEditingCtx := Context{Page: "Services"}
	catalog = Catalog(notEditingCtx)

	editor = scopeTitled(t, catalog, "Editor")
	for _, binding := range []key.Binding{Editor.Indent, Editor.Outdent} {
		if entryIn(t, editor, binding).Available {
			t.Errorf("%q should be dimmed when not editing", binding.Help().Key)
		}
	}
}

// The alt chords are the aliases the footer has no room for; the overlay is
// where they are advertised, derived from the labels.
func TestCatalogListsTheChordAliases(t *testing.T) {
	catalog := Catalog(Context{Page: "Home"})

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

// The ungrouped row's 'A' verb flips with its materialization: adopt while
// the row is derived, release once a real ungrouped profile backs it. The
// other face is dimmed, and neither is offered on a real group.
func TestUngroupedAdoptReleaseAvailability(t *testing.T) {
	derived := Catalog(Context{Page: "Home", ReadOnlyGroup: true})
	groups := scopeTitled(t, derived, "Groups")
	if !entryIn(t, groups, List.AdoptUngrouped).Available {
		t.Error("A adopt should be available on the derived ungrouped row")
	}
	if entryIn(t, groups, List.ReleaseUngrouped).Available {
		t.Error("A release should be dimmed while the row is derived")
	}

	materialized := Catalog(Context{Page: "Home", ReadOnlyGroup: true, UngroupedMaterialized: true})
	groups = scopeTitled(t, materialized, "Groups")
	if !entryIn(t, groups, List.ReleaseUngrouped).Available {
		t.Error("A release should be available on the materialized ungrouped row")
	}
	if entryIn(t, groups, List.AdoptUngrouped).Available {
		t.Error("A adopt should be dimmed while the row is materialized")
	}

	realGroup := Catalog(Context{Page: "Home"})
	groups = scopeTitled(t, realGroup, "Groups")
	if entryIn(t, groups, List.AdoptUngrouped).Available {
		t.Error("A adopt should be dimmed on a real group")
	}
	if entryIn(t, groups, List.ReleaseUngrouped).Available {
		t.Error("A release should be dimmed on a real group")
	}
}

// A binding identity is keystrokes plus help text: the overlay's dimming is
// only as good as this comparison.
func TestSameBinding(t *testing.T) {
	if !sameBinding(List.Delete, List.Delete) {
		t.Error("a binding should equal itself")
	}
	if sameBinding(List.Delete, List.Filter) {
		t.Error("different bindings should not compare equal")
	}
	// ClearFilter and Cancel share esc but say different things.
	if sameBinding(List.ClearFilter, Overlay.Cancel) {
		t.Error("same key with different help is a different row")
	}
}

// The Backups page is the one page with focus left, and which half holds it
// changes exactly one thing about the keys: whether the arrows navigate the
// list or scroll the preview. Restore is live either way, because focus moves
// the arrows and not the selection.
func TestBackupsFocusSwapsTheArrowHint(t *testing.T) {
	list := Active(Context{Page: "Backups"})
	if !containsBinding(list, Backup.Navigate) {
		t.Error("the list-focused footer should offer navigate")
	}
	if containsBinding(list, Backup.Scroll) {
		t.Error("the list-focused footer should not offer scroll")
	}

	preview := Active(Context{Page: "Backups", BackupsFocus: apptypes.BackupsPreview})
	if !containsBinding(preview, Backup.Scroll) {
		t.Error("the preview-focused footer should offer scroll")
	}
	if containsBinding(preview, Backup.Navigate) {
		t.Error("the preview-focused footer should not offer navigate")
	}

	for _, bindings := range [][]key.Binding{list, preview} {
		if !containsBinding(bindings, Backup.Restore) {
			t.Error("restore should be live on either half of the page")
		}
	}
}

// tab is live on Backups and nowhere else on a body page, so it is advertised
// there and only there. Its twin shift+tab has no footer slot of its own but
// is pressable wherever tab is.
func TestTabIsOfferedOnBackupsOnly(t *testing.T) {
	if !containsBinding(Active(Context{Page: "Backups"}), Global.NextPanel) {
		t.Error("the Backups footer should offer tab: it is the one page with two focus stops")
	}

	for _, page := range []apptypes.Page{apptypes.PageHome, apptypes.PageServices, apptypes.PageComposeFiles} {
		if containsBinding(Active(Context{Page: page}), Global.NextPanel) {
			t.Errorf("%s offers tab, which is dead there", page)
		}
	}

	backups := Catalog(Context{Page: "Backups"})
	global := scopeTitled(t, backups, "Global")
	for _, binding := range []key.Binding{Global.NextPanel, Global.PrevPanel} {
		if !entryIn(t, global, binding).Available {
			t.Errorf("%q should be live in the overlay on the Backups page", binding.Help().Key)
		}
	}
}

// esc does nothing on the Backups page - there is no selection to clear and no
// filter to abandon - so it is not offered. The bar does not advertise inert
// keys.
func TestBackupsDoesNotOfferBack(t *testing.T) {
	if containsBinding(Active(Context{Page: "Backups"}), Global.Back) {
		t.Error("the Backups footer offers esc back, which does nothing on that page")
	}
}

// Restore is r alone. enter was dropped: it is too easy to hit by reflex while
// navigating, for an action that overwrites a live file.
func TestRestoreIsBoundToROnly(t *testing.T) {
	if got := Backup.Restore.Keys(); len(got) != 1 || got[0] != "r" {
		t.Errorf("the restore binding carries %v, want r alone", got)
	}
}
