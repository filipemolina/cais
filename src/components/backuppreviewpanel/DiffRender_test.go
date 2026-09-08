package backuppreviewpanel

import (
	"fmt"
	"image/color"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/utils"
)

// seedCopy writes contents to a fresh temp file, snapshots it, and returns
// the store's entry for that copy. The entry is found by its content hash,
// never by position: two snapshots inside the same second tie on the name's
// timestamp prefix and sort by hash, which is not write order.
func seedCopy(tb testing.TB, filename, contents string) utils.BackupEntry {
	tb.Helper()

	path := filepath.Join(tb.TempDir(), filename)
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		tb.Fatalf("writing %s: %v", filename, err)
	}
	if err := utils.SnapshotFile(path); err != nil {
		tb.Fatalf("snapshot %s: %v", filename, err)
	}

	entries, err := utils.ListBackups(path)
	if err != nil {
		tb.Fatalf("ListBackups(%s): %v", path, err)
	}

	want := utils.ContentSHA8([]byte(contents))
	for _, entry := range entries {
		if entry.SHA8 == want {
			return entry
		}
	}
	tb.Fatalf("seeded copy %s not in the store", want)
	return utils.BackupEntry{}
}

// diffedPanel drives a panel the way the running app does for a copy whose
// live file holds liveContents: layout, list read, selection, read back.
// The list half of the read comes from Model_test.go's listMsgFor.
func diffedPanel(t *testing.T, source, copy, live string) Model {
	t.Helper()

	entry := seedCopy(t, composeNameFor(source), copy)

	updated, _ := New().Update(cmds.SetBodyLayoutMsg{LeftWidth: 60, RightWidth: 60, Height: 30})
	m := updated.(Model)

	updated, _ = m.Update(listMsgFor(nil, source, live))
	m = updated.(Model)

	return selectAndLoad(t, m, entry)
}

func composeNameFor(source string) string {
	if source == ".env" {
		return ".env"
	}
	return "compose.yaml"
}

// vpDims mirrors the sizing Model.setSize does, so the tests can build the
// exact strings the frame should carry: the viewport takes the panel minus
// the wrapper's frame, and the gutter takes two more columns off the wash's
// width.
func vpDims(panelWidth, panelHeight int) (vpWidth, vpHeight int) {
	frameW, frameH := chrome.WrapperStyle.GetFrameSize()
	return max(1, panelWidth-frameW), max(1, panelHeight-frameH-2)
}

// framed sizes an already-loaded panel and returns its rendered frame.
func framed(t *testing.T, m Model) string {
	t.Helper()

	sized, _ := m.Update(cmds.SetBodyLayoutMsg{LeftWidth: 60, RightWidth: 60, Height: 30})
	return sized.(Model).View().Content
}

// styledRun renders text in fg on bg (either may be nil) and drops the
// trailing reset: the panel's FillBackground pass re-asserts the panel
// background after every reset, so an expectation that continues past one
// is not a substring of the frame even when the rendering is right.
func styledRun(text string, fg, bg color.Color) string {
	style := lipgloss.NewStyle()
	if fg != nil {
		style = style.Foreground(fg)
	}
	if bg != nil {
		style = style.Background(bg)
	}
	return strings.TrimSuffix(style.Render(text), ansi.ResetStyle)
}

// A diff renders as git does: the changed lines washed in the theme's add/
// remove colors with a +/- gutter marker, the unchanged lines keeping the
// copy's own rendering. Everything below - the empty state, the auto-scroll,
// the bleed seal - depends on this shape being right.
func TestTheDiffRendersWashesAndMarkers(t *testing.T) {
	m := diffedPanel(t, "compose",
		"services:\n  x: 1\n  y: 2\n",
		"services:\n  x: 9\n  y: 2\n")

	frame := framed(t, m)
	theme := appstyles.Active

	// Content width: the viewport takes the panel minus its frame, and the
	// gutter takes two more columns; the wash pads to exactly that.
	vpWidth, _ := vpDims(60, 30)
	contentWidth := max(1, vpWidth-2)

	// The changed lines' text sits on the wash. The direction is the
	// restore's: the copy's "x: 1" is what restoring adds (Insert, add
	// wash), the live file's "x: 9" what it removes (Delete, remove wash).
	if !strings.Contains(frame, styledRun("  x: 9", theme.TextPrimary, theme.DiffRemove)) {
		t.Errorf("the line restoring removes is not washed in the remove color:\n%s", frame)
	}
	if !strings.Contains(frame, styledRun("  x: 1", theme.TextPrimary, theme.DiffAdd)) {
		t.Errorf("the line restoring adds is not washed in the add color:\n%s", frame)
	}

	// ...and the wash pads past the text to the row's full width, so a
	// changed line reads as a row and not as colored text on a plain panel.
	pad := strings.Repeat(" ", min(20, max(1, contentWidth-6)))
	if !strings.Contains(frame, styledRun(pad, nil, theme.DiffRemove)) {
		t.Errorf("the remove wash stops at the text instead of filling the row:\n%s", frame)
	}
	if !strings.Contains(frame, styledRun(pad, nil, theme.DiffAdd)) {
		t.Errorf("the add wash stops at the text instead of filling the row:\n%s", frame)
	}

	// The gutter marker sits in its wash, in the status color the wash was
	// blended from.
	if !strings.Contains(frame, styledRun("- ", theme.Danger, theme.DiffRemove)) {
		t.Error("the deleted line has no - gutter marker in the remove wash")
	}
	if !strings.Contains(frame, styledRun("+ ", theme.StatusRunning, theme.DiffAdd)) {
		t.Error("the inserted line has no + gutter marker in the add wash")
	}

	// Unchanged lines keep the copy's own YAML rendering: the mapping key is
	// accent-colored exactly as the plain preview draws it.
	if !strings.Contains(frame, lipgloss.NewStyle().Foreground(theme.Accent).Render("services")) {
		t.Errorf("the equal lines lost their YAML highlighting:\n%s", frame)
	}

	// ...and the changed lines lost theirs - the wash replaces the syntax
	// colors on them, because a colored key inside a colored row is two
	// kinds of signal fighting for one line.
	if strings.Contains(frame, lipgloss.NewStyle().Foreground(theme.Accent).Render("x")) {
		t.Errorf("a changed line still carries the YAML key color:\n%s", frame)
	}
}

// The .env contract survives the diff: the preview is the exact bytes a
// restore would write, secrets included. DESIGN.md holds this as a standing
// constraint, so a change that starts masking values breaks here on purpose.
func TestAnEnvDiffShowsItsSecrets(t *testing.T) {
	m := diffedPanel(t, ".env",
		"A=1\nSECRET=hunter2\n",
		"A=1\nB=2\n")

	frame := framed(t, m)

	if !strings.Contains(ansi.Strip(frame), "SECRET=hunter2") {
		t.Errorf("the .env diff masked its secret values:\n%s", ansi.Strip(frame))
	}

	theme := appstyles.Active
	if !strings.Contains(frame, styledRun("SECRET=hunter2", theme.TextPrimary, theme.DiffAdd)) {
		t.Errorf("the secret line restored by this copy is not washed in the add color:\n%s", frame)
	}
}

// A copy whose bytes are the live file's gets the identical empty state, not
// a rendered diff of nothing. It is read off the computed lines, so the
// negative control is the same copy with no live bytes on the message: no
// diff, no claim.
func TestAnIdenticalCopyGetsTheEmptyState(t *testing.T) {
	entry := seedCopy(t, "compose.yaml", "services:\n  app:\n    image: ONLY-IN-COPY\n")

	updated, _ := New().Update(cmds.SetBodyLayoutMsg{LeftWidth: 60, RightWidth: 60, Height: 30})
	m := updated.(Model)

	updated, _ = m.Update(listMsgFor(nil, "compose", "services:\n  app:\n    image: ONLY-IN-COPY\n"))
	m = updated.(Model)
	m = selectAndLoad(t, m, entry)

	frame := framed(t, m)
	if !strings.Contains(frame, "Identical to the live file") {
		t.Errorf("an identical copy shows no empty state:\n%s", frame)
	}
	if strings.Contains(frame, "ONLY-IN-COPY") {
		t.Errorf("the identical empty state still renders the file:\n%s", frame)
	}

	// Without the live bytes there is no diff, so there is no answer to
	// give - the copy's own bytes show, as they did before the diff existed.
	plain := seedCopy(t, "compose.yaml", "services:\n  app:\n    image: ONLY-IN-COPY\n")
	bare := selectAndLoad(t, New().(Model), plain)
	bareFrame := framed(t, bare)
	if strings.Contains(bareFrame, "Identical to the live file") {
		t.Error("a copy with no diff available claimed to be identical")
	}
	if !strings.Contains(ansi.Strip(bareFrame), "ONLY-IN-COPY") {
		t.Errorf("the fallback stopped showing the copy's own bytes:\n%s", ansi.Strip(bareFrame))
	}
}

// The diff opens with the first change mid-screen, so the user lands on the
// part that matters instead of on the file's header lines. SoftWrap is off,
// so the line index maps 1:1 to a viewport row.
func TestTheDiffOpensOnTheFirstChange(t *testing.T) {
	live := numberedLines(40, func(i int) string { return fmt.Sprintf("l%02d", i) })
	copyLines := numberedLines(40, func(i int) string { return fmt.Sprintf("l%02d", i) })
	copyLines[20] = "CHANGED"

	m := diffedPanel(t, "compose", strings.Join(copyLines, "\n")+"\n", strings.Join(live, "\n")+"\n")

	// The change sits half a viewport above mid-screen; the file is long
	// enough that the offset is not clamped.
	_, vpHeight := vpDims(60, 30)
	if got, want := m.vp.YOffset(), 20-vpHeight/2; got != want {
		t.Errorf("the diff opened on line %d, want %d (first change mid-screen)", got, want)
	}

	// A change near the top clamps to the top rather than over-shooting
	// into a negative offset.
	early := append([]string(nil), live...)
	early[2] = "CHANGED-EARLY"
	earlyCopy := append([]string(nil), copyLines...)
	earlyCopy[20] = "l20"
	mEarly := diffedPanel(t, "compose", strings.Join(earlyCopy, "\n")+"\n", strings.Join(early, "\n")+"\n")
	if got := mEarly.vp.YOffset(); got != 0 {
		t.Errorf("an early change scrolled to line %d, want the top", got)
	}
}

// A re-list after a write replaces the live side without a new selection -
// and the scroll follows the new diff's first change, or the panel keeps
// opening on a change that is no longer the first one.
func TestARelistRescrollsToTheNewFirstChange(t *testing.T) {
	live := numberedLines(40, func(i int) string { return fmt.Sprintf("l%02d", i) })
	copyLines := numberedLines(40, func(i int) string { return fmt.Sprintf("l%02d", i) })
	copyLines[20] = "CHANGED"

	entry := seedCopy(t, "compose.yaml", strings.Join(copyLines, "\n")+"\n")

	updated, _ := New().Update(cmds.SetBodyLayoutMsg{LeftWidth: 60, RightWidth: 60, Height: 30})
	m := updated.(Model)
	updated, _ = m.Update(listMsgFor(nil, "compose", strings.Join(live, "\n")+"\n"))
	m = updated.(Model)
	m = selectAndLoad(t, m, entry)

	if got := m.vp.YOffset(); got == 0 {
		t.Fatal("precondition: the diff did not scroll to its first change")
	}

	// The live file moves so the first change is line 3 - the diff, and the
	// scroll with it, must follow without any new selection publish. Near
	// the top the offset clamps back to it.
	newLive := append([]string(nil), copyLines...)
	newLive[3] = "CHANGED-EARLIER"
	updated, _ = m.Update(listMsgFor(nil, "compose", strings.Join(newLive, "\n")+"\n"))
	m = updated.(Model)

	if got := m.vp.YOffset(); got != 0 {
		t.Errorf("after the re-list the diff opens on line %d, want the top", got)
	}
}

// A re-list can also take the live bytes away (the source left the config).
// The panel must fall back to the copy's own bytes, not keep rendering a
// diff against a file that no longer exists.
func TestLosingTheLiveSideFallsBackToTheCopy(t *testing.T) {
	m := diffedPanel(t, "compose", "services:\n  x: 1\n", "services:\n  x: 9\n")

	updated, _ := m.Update(cmds.BackupListMsg{Live: []utils.LiveSource{}})
	m = updated.(Model)

	if m.lines != nil {
		t.Errorf("a list read with no live bytes left a diff behind: %+v", m.lines)
	}

	frame := ansi.Strip(framed(t, m))
	if strings.Contains(frame, "x: 9") {
		t.Errorf("the fallback still renders the removed live line:\n%s", frame)
	}
	if !strings.Contains(frame, "x: 1") {
		t.Errorf("the fallback stopped showing the copy's own bytes:\n%s", frame)
	}
}

// The washes are painted per line, so the invariant the app seals everywhere
// else holds here too: no run of spaces may render on the terminal's own
// default background. It is checked on a fully rendered frame, per theme,
// because a wash that leaves a hole is theme-specific by definition.
func TestTheDiffFrameIsSealedAgainstBleeds(t *testing.T) {
	original := appstyles.Active
	t.Cleanup(func() { appstyles.Active = original })

	m := diffedPanel(t, "compose",
		"services:\n  x: 1\n  y: 2\n",
		"services:\n  x: 9\n  y: 2\n")

	for _, name := range slices.Sorted(maps.Keys(appstyles.Themes)) {
		if !appstyles.SetTheme(name) {
			t.Fatalf("theme %q is not registered", name)
		}

		frame := framed(t, m)
		if appstyles.HasBackgroundBleed(frame) {
			t.Errorf("%s: the diff frame bleeds to the terminal's default background", name)
		}
	}
}

// A file edited on Windows carries CRLF endings, and the engine trims only
// the trailing \n - a \r would survive into every rendered row, where the
// wash padding drawn after it repaints the row from column 0, over its own
// text. The plain preview never saw this because its \r was last on the
// row; both sides of the diff are normalized on arrival instead.
func TestACRFLFileRendersWithoutCarriageReturns(t *testing.T) {
	m := diffedPanel(t, "compose",
		"services:\r\n  x: 1\r\n",
		"services:\r\n  x: 9\r\n")

	frame := framed(t, m)
	if strings.ContainsRune(frame, '\r') {
		t.Errorf("the diff frame carries a carriage return into the terminal:\n%s", frame)
	}

	// And the diff still aligns: the changed line is washed and marked on
	// one row, not split by a stray \r.
	if !strings.Contains(ansi.Strip(frame), "-   x: 9") {
		t.Errorf("the CRLF file's diff lost its gutter marker or content:\n%s", ansi.Strip(frame))
	}
}

// numberedLines builds n lines through f, which the auto-scroll fixtures
// differ at one index.
func numberedLines(n int, f func(int) string) []string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = f(i)
	}
	return lines
}

// benchDiffPanel builds a loaded panel showing the diff of a compose file at
// the preview's own ceiling: a whole-file diff of a 1000-line compose file
// with a change every 25th line. Setup is outside the loops; everything
// per-keystroke is inside them.
func benchDiffPanel(b *testing.B) (Model, utils.BackupEntry) {
	b.Helper()

	live := numberedLines(1000, func(i int) string { return fmt.Sprintf("key-%04d: value", i) })
	copyLines := append([]string(nil), live...)
	for i := 0; i < len(copyLines); i += 25 {
		copyLines[i] = "key-changed: other value"
	}

	entry := seedCopy(b, "compose.yaml", strings.Join(copyLines, "\n")+"\n")

	sized, _ := New().Update(cmds.SetBodyLayoutMsg{LeftWidth: 60, RightWidth: 60, Height: 30})
	m := sized.(Model)
	updated, _ := m.Update(listMsgFor(nil, "compose", strings.Join(live, "\n")+"\n"))
	m = updated.(Model)
	m = selectAndLoad(b, m, entry)

	return m, entry
}

// The diff is styled per line per frame through the viewport's hooks, so the
// plan's trap about per-keystroke rebuilds applies here and is measured
// rather than assumed: a cursor move rebuilds the content (selection, read,
// recompute), a page keystroke only re-renders a frame. Neither may grow
// with the file in a way that eats the frame budget.
func BenchmarkTheDiffRenders(b *testing.B) {
	m, entry := benchDiffPanel(b)

	b.Run("cursor-move", func(b *testing.B) {
		for b.Loop() {
			next, cmd := m.Update(cmds.SetSelectedBackupMsg(entry))
			if cmd == nil {
				b.Fatal("selecting a copy issued no read")
			}
			moved, _ := next.(Model).Update(cmd())
			m = moved.(Model)
			_ = m.View()
		}
	})

	b.Run("page", func(b *testing.B) {
		for b.Loop() {
			next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
			m = next.(Model)
			_ = m.View()
		}
	})
}
