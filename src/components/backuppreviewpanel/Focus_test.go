package backuppreviewpanel

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/utils"
)

// yOffset is the line the viewport is showing first, and atBottom whether it
// has reached the end. viewport.Model's accessors are pointer methods, so the
// panel has to land in a variable before either can be asked.
func yOffset(m tea.Model) int {
	panel := m.(Model)

	return panel.vp.YOffset()
}

func atBottom(m tea.Model) bool {
	panel := m.(Model)

	return panel.vp.AtBottom()
}

// scrollable is a sized panel holding a copy far taller than its viewport, so
// there is somewhere for a scroll to go. The bytes come from a synthetic
// entry rather than a real .bak: the read is short-circuited by feeding the
// contents message directly, and what is under test is the routing of keys,
// not the read.
func scrollable(t *testing.T, focus apptypes.BackupsFocus) Model {
	t.Helper()

	entry := utils.BackupEntry{
		Source: "compose",
		Name:   "20260811T091500.0000000a.bak",
		File:   "compose.yaml",
		SHA8:   "0000000a",
		Path:   "/nowhere/tall.bak",
	}

	updated, _ := New().Update(cmds.SetBodyLayoutMsg{LeftWidth: 60, RightWidth: 60, Height: 24})
	m := updated.(Model)

	updated, _ = m.Update(cmds.SetSelectedBackupMsg(entry))
	m = updated.(Model)

	updated, _ = m.Update(backupContentsMsg{
		Path:     entry.Path,
		Source:   entry.Source,
		Contents: []byte(strings.Repeat("services:\n", 300)),
	})
	m = updated.(Model)

	updated, _ = m.Update(cmds.SetBackupsFocusMsg(focus))
	m = updated.(Model)

	if m.vp.TotalLineCount() <= m.vp.Height() {
		t.Fatalf("the seeded copy fits in the viewport (%d lines in %d rows); nothing to scroll",
			m.vp.TotalLineCount(), m.vp.Height())
	}

	return m
}

// The preview starts unfocused. The page opens on the list, and there is
// nothing here to scroll until the cursor has picked a row.
func TestPreviewStartsUnfocused(t *testing.T) {
	if New().(Model).focused {
		t.Error("the preview should not hold focus when the page opens")
	}
}

// Unfocused, the arrows belong to the list. Both panels receive every
// keystroke, so the preview has to decline the ones it does not own or one
// arrow press scrolls the file *and* moves the cursor.
func TestUnfocusedPreviewIgnoresTheArrows(t *testing.T) {
	m := scrollable(t, apptypes.BackupsList)

	for _, msg := range []tea.KeyPressMsg{
		{Code: tea.KeyDown},
		{Code: 'j', Text: "j"},
		{Code: tea.KeyPgDown},
		{Code: 'd', Mod: tea.ModCtrl},
		{Code: 'G', Text: "G"},
		{Code: tea.KeyEnd},
	} {
		moved, _ := m.Update(msg)

		if got := yOffset(moved); got != 0 {
			t.Errorf("%v scrolled the preview to line %d while the list held focus", msg, got)
		}
	}

	// Negative control: the same keys scroll once focus arrives, so the
	// assertions above are about focus rather than about a viewport that never
	// had anywhere to scroll to.
	focused, _ := m.Update(cmds.SetBackupsFocusMsg(apptypes.BackupsPreview))
	scrolled, _ := focused.(Model).Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if yOffset(scrolled) == 0 {
		t.Error("↓ did not scroll the preview once it held focus")
	}
}

// pgdn and ctrl+d reach the viewport's own keymap while the preview is
// focused - the dead map installed in New() finally has keys routed to it.
func TestFocusedPreviewPagesTheFile(t *testing.T) {
	m := scrollable(t, apptypes.BackupsPreview)

	paged, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	full := yOffset(paged)
	if full == 0 {
		t.Fatal("pgdn did not page the focused preview")
	}

	halved, _ := m.Update(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	half := yOffset(halved)
	if half == 0 || half >= full {
		t.Errorf("ctrl+d moved %d lines, want a positive half of pgdn's %d", half, full)
	}
}

// G and end reach the bottom of the file, and g and home come back. The
// viewport's keymap has no top/bottom binding at all, so without the panel
// matching these itself the preview would be the one panel where G does
// nothing while the list beside it jumps to its last row.
func TestFocusedPreviewJumpsToBothEnds(t *testing.T) {
	m := scrollable(t, apptypes.BackupsPreview)

	for _, jump := range []struct {
		name      string
		toBottom  tea.KeyPressMsg
		backToTop tea.KeyPressMsg
	}{
		{"G/g", tea.KeyPressMsg{Code: 'G', Text: "G"}, tea.KeyPressMsg{Code: 'g', Text: "g"}},
		{"end/home", tea.KeyPressMsg{Code: tea.KeyEnd}, tea.KeyPressMsg{Code: tea.KeyHome}},
	} {
		bottom, _ := m.Update(jump.toBottom)
		if !atBottom(bottom) {
			t.Errorf("%s: the first key did not reach the bottom of the file (it is on line %d)",
				jump.name, yOffset(bottom))
		}

		top, _ := bottom.(Model).Update(jump.backToTop)
		if got := yOffset(top); got != 0 {
			t.Errorf("%s: the second key left the file on line %d, want the top", jump.name, got)
		}
	}
}

// r is not the preview's key even while it holds focus: the list owns the
// cursor and therefore owns the restore, and it receives the same keystroke.
// A preview that answered r would give the page two handlers for one verb.
func TestFocusedPreviewDoesNotClaimRestore(t *testing.T) {
	m := scrollable(t, apptypes.BackupsPreview)

	before := yOffset(m)
	after, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})

	if cmd != nil {
		t.Errorf("the preview answered r with %T", cmd())
	}
	if got := yOffset(after); got != before {
		t.Errorf("r scrolled the preview from line %d to %d", before, got)
	}
}
