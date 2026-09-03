package keys

import (
	"testing"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
)

// The keys a library default claims that this app spends elsewhere. Each is
// checked against the replacement maps below: a key in this set must not be
// bound in a map the app installs, or the component answering it would be
// answering a key it never declared.
//
// See ListKeyMap, ModalListKeyMap and ReadOnlyViewportKeyMap for why each
// map exists; this test is the guard that keeps a future edit from quietly
// handing one of these keys back to a child component.
var contestedKeys = []string{"q", "?", "b", "u", "f", "d", "ctrl+c"}

func boundKeys(bindings ...key.Binding) map[string]bool {
	out := map[string]bool{}
	for _, b := range bindings {
		for _, k := range b.Keys() {
			out[k] = true
		}
	}
	return out
}

func listMapKeys(m list.KeyMap) map[string]bool {
	return boundKeys(
		m.CursorUp, m.CursorDown, m.NextPage, m.PrevPage,
		m.GoToStart, m.GoToEnd, m.Filter, m.ClearFilter,
		m.CancelWhileFiltering, m.AcceptWhileFiltering,
		m.ShowFullHelp, m.CloseFullHelp, m.Quit, m.ForceQuit,
	)
}

func viewportMapKeys(m viewport.KeyMap) map[string]bool {
	return boundKeys(
		m.PageDown, m.PageUp, m.HalfPageUp, m.HalfPageDown,
		m.Up, m.Down, m.Left, m.Right,
	)
}

// TestLibraryDefaultsClaimContestedKeys is the premise the other two tests
// rest on. If bubbles ever stops binding these, the replacements below are
// no longer load-bearing and this whole file should be revisited rather than
// left passing vacuously.
func TestLibraryDefaultsClaimContestedKeys(t *testing.T) {
	defaults := listMapKeys(list.DefaultKeyMap())
	for _, k := range contestedKeys {
		if !defaults[k] {
			t.Errorf("list.DefaultKeyMap no longer binds %q - revisit ListKeyMap and ModalListKeyMap", k)
		}
	}

	vp := viewportMapKeys(viewport.DefaultKeyMap())
	for _, k := range []string{"b", "u", "f", "d", "h", "l"} {
		if !vp[k] {
			t.Errorf("viewport.DefaultKeyMap no longer binds %q - revisit ReadOnlyViewportKeyMap", k)
		}
	}
}

func TestListKeyMapsLeaveContestedKeysAlone(t *testing.T) {
	maps := map[string]list.KeyMap{
		"ListKeyMap":      ListKeyMap(),
		"ModalListKeyMap": ModalListKeyMap(),
	}

	for name, m := range maps {
		t.Run(name, func(t *testing.T) {
			bound := listMapKeys(m)
			for _, k := range contestedKeys {
				if bound[k] {
					t.Errorf("%s binds %q, which the app spends elsewhere", name, k)
				}
			}
			// A map that moves the cursor is the reason to install it at
			// all; an empty one would pass the check above vacuously.
			if !bound["up"] || !bound["down"] {
				t.Errorf("%s does not move the cursor", name)
			}
		})
	}
}

func TestReadOnlyViewportKeyMapLeavesLettersAlone(t *testing.T) {
	bound := viewportMapKeys(ReadOnlyViewportKeyMap())

	// f is the sharpest of these: Overlay.Follow in the logs modal and
	// PageDown in the viewport default.
	for _, k := range []string{"b", "u", "f", "d", "h", "l"} {
		if bound[k] {
			t.Errorf("ReadOnlyViewportKeyMap binds %q, which the app spends elsewhere", k)
		}
	}

	if !bound["pgup"] || !bound["pgdown"] {
		t.Error("ReadOnlyViewportKeyMap does not page")
	}
}

// TestModalListKeyMapUnbindsFilter guards the reason Filter is unbound
// rather than switched off with SetFilteringEnabled: every modal that
// installs this map hides its title row, and the filter input renders into
// that row, so a live filter would swallow keystrokes with nothing on
// screen to show for them.
func TestModalListKeyMapUnbindsFilter(t *testing.T) {
	m := ModalListKeyMap()

	for name, b := range map[string]key.Binding{
		"Filter":               m.Filter,
		"ClearFilter":          m.ClearFilter,
		"CancelWhileFiltering": m.CancelWhileFiltering,
		"AcceptWhileFiltering": m.AcceptWhileFiltering,
	} {
		if len(b.Keys()) != 0 {
			t.Errorf("ModalListKeyMap.%s binds %v, want no keys", name, b.Keys())
		}
	}
}
