package backuppreviewpanel

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/diff"
	"github.com/filipemolina/cais/src/highlight"
)

func (m Model) View() tea.View {
	// The panel's whole tier moves with focus, frame included, rather than a
	// heavier border appearing - see chrome.PanelBgFor.
	bg := chrome.PanelBgFor(m.focused)

	bodyWidth := max(1, chrome.PanelBodyWidth(m.panelWidth))
	bodyAvail := max(1, chrome.PanelBodyHeight(m.panelHeight))

	var body string
	switch {
	case !m.hasSelection():
		body = chrome.EmptyCard(bodyWidth, bodyAvail, bg,
			"Nothing selected",
			"Pick a version on the left to see what it holds.",
			"", "")

	case m.loadErr != nil:
		body = chrome.EmptyCard(bodyWidth, bodyAvail, bg,
			"Could not read this copy",
			m.loadErr.Error(),
			"", "")

	case m.identicalToLive():
		// The diff is the page's answer to "what would restoring change?",
		// so a diff with no changes is a state of its own, not a file
		// rendered for nothing: restoring would write the same bytes back.
		// The list's (current) marker says the same thing from the shas;
		// this card says it where the user is looking.
		body = chrome.EmptyCard(bodyWidth, bodyAvail, bg,
			"Identical to the live file",
			"Restoring this copy would change nothing.",
			"", "")

	default:
		body = m.renderContent(bodyWidth, bodyAvail, bg)
	}

	titleRight := ""
	if m.hasSelection() {
		titleRight = lipgloss.NewStyle().
			Foreground(appstyles.Active.TextDim).
			Render(fmt.Sprintf("%s · %s", m.entry.File, m.entry.SHA8))
	}

	screen := chrome.PanelFrameOn("Preview", titleRight, m.panelWidth, m.panelHeight, bg, body)
	return tea.NewView(screen)
}

// renderContent puts the viewport in the panel body under a header naming
// the copy's write time, so the panel says which version is on screen
// without the reader having to look back at the list.
func (m Model) renderContent(width, avail int, bg color.Color) string {
	header := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextDim).
		Background(bg).
		Width(width).
		Render(m.entry.Timestamp.UTC().Format("2006-01-02 15:04") + " UTC")

	vp := m.vp
	if m.lines != nil {
		// The hooks are installed on a per-render copy of the viewport and
		// close over this frame's model, so they can never outlive the
		// lines they read: a value model's Update return is a new copy, and
		// a hook kept on m.vp would read through a stale one.
		vp.StyleLineFunc = func(i int) lipgloss.Style {
			return m.lineStyle(i)
		}
		vp.LeftGutterFunc = func(gc viewport.GutterContext) string {
			return m.gutterMarker(gc.Index)
		}
	}

	// Clip the viewport to the height left under the header row.
	vpView := appstyles.FillBackground(bg, vp.View())
	vpView = lipgloss.NewStyle().MaxHeight(max(0, avail-1)).Render(vpView)

	content := lipgloss.JoinVertical(lipgloss.Left, header, vpView)
	return chrome.PanelBodyWithFooter(width, avail, bg, content, "")
}

// lineStyle is the viewport's per-line style while a diff is showing: the
// theme's wash behind the changed lines, nothing for the equal ones.
//
// The wash replaces the syntax colors on a changed line rather than sitting
// under them - the line's meaning is "this changed", and a YAML key colored
// inside a colored row is two kinds of signal fighting for one line. The
// equal lines pass through the zero style untouched, which is what keeps
// the copy's own rendering (YAML for compose, plain text for .env) intact.
//
// Width pads the wash to the full row so a changed line reads as a row, not
// as colored text on a plain panel; the width is the viewport's content
// area, which the viewport itself cuts horizontal scrolling to, so the pad
// survives the cut.
func (m Model) lineStyle(i int) lipgloss.Style {
	if i < 0 || i >= len(m.lines) || m.lines[i].Kind == diff.Equal {
		return lipgloss.Style{}
	}

	wash := appstyles.Active.DiffAdd
	if m.lines[i].Kind == diff.Delete {
		wash = appstyles.Active.DiffRemove
	}

	return lipgloss.NewStyle().
		Foreground(appstyles.Active.TextPrimary).
		Background(wash).
		Width(max(1, m.vp.Width()-2))
}

// gutterMarker draws the +/- column the git-style diff reads by: `+` for
// content restoring would add to the live file, `-` for content it would
// remove, blanks for what both agree on. It is drawn by the viewport's
// gutter hook so it survives horizontal scrolling, and the marker cell sits
// in the wash with the line's text so a changed row reads as one row.
//
// The glyph takes the status color the wash was blended from, not the wash
// itself - DiffRemove on DiffRemove is one color, an invisible marker. The
// status color on its wash is the pair TestWCAGContrastAgainstSurfaces
// holds to 2.4, and it is what keeps the meaning legible where the wash is
// too subtle to see.
//
// The marker text is always two columns wide - the gutter's real width has
// to be consistent or the viewport's own width math drifts.
func (m Model) gutterMarker(i int) string {
	if i < 0 || i >= len(m.lines) {
		return "  "
	}

	theme := appstyles.Active
	switch m.lines[i].Kind {
	case diff.Insert:
		return lipgloss.NewStyle().Foreground(theme.StatusRunning).Background(theme.DiffAdd).Render("+ ")
	case diff.Delete:
		return lipgloss.NewStyle().Foreground(theme.Danger).Background(theme.DiffRemove).Render("- ")
	default:
		return "  "
	}
}

// diffContent builds the viewport's text for a diff: one string per diff
// line, in order. Equal lines take their rendering from the copy's own
// styled block, consumed in step, so the YAML colorizer's block-scalar state
// (which spans lines) is carried over exactly as the plain preview drew it.
// Walking Equal+Insert in order reconstructs the copy byte for byte, so the
// two sequences stay aligned by construction - which is why an Insert
// consumes a slot even though its rendering is dropped: a changed line
// renders raw, because the wash replaces the syntax colors on it (see
// lineStyle) and letting the colorizer's runs through would put a colored
// key inside a colored row. The bounds guard is defense against a renderer
// and a diff ever disagreeing, and degrades those lines to their raw text
// rather than panicking.
func diffContent(source, raw string, lines []diff.Line) []string {
	styled := strings.Split(renderCopy(source, raw), "\n")

	out := make([]string, 0, len(lines))
	k := 0
	for _, line := range lines {
		switch line.Kind {
		case diff.Equal:
			if k < len(styled) {
				out = append(out, styled[k])
			} else {
				out = append(out, line.Content)
			}
			k++
		case diff.Insert:
			k++
			out = append(out, line.Content)
		default:
			out = append(out, line.Content)
		}
	}
	return out
}

// renderCopy renders the copy's own bytes the way the panel has always shown
// them: compose copies through the YAML colorizer, .env copies raw with an
// explicit foreground. The contract underneath is docs/DESIGN.md's: the
// preview is the exact bytes a restore would write, secrets included - the
// diff view inherits it, so an .env diff shows secret values in plain text
// too.
func renderCopy(source, raw string) string {
	if source == "compose" {
		return highlight.YAML(raw)
	}
	return plainText(raw)
}

// plainText gives unhighlighted content an explicit foreground.
//
// Without one the text carries no SGR at all and lands on the terminal's own
// default foreground, which has nothing to do with the active theme: on a
// light theme the .env preview was pale grey on a pale panel and effectively
// unreadable. Every other body of text in the app names its colour, and
// highlight.YAML does it per line for compose copies; this is the same for the
// copies that get no syntax highlighting.
//
// Styled per line rather than over the whole block because lipgloss closes
// each styled run with a reset, so a single Render of a multi-line string
// leaves everything after the first newline unstyled again.
func plainText(content string) string {
	if content == "" {
		return content
	}

	style := lipgloss.NewStyle().Foreground(appstyles.Active.TextPrimary)

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		lines[i] = style.Render(line)
	}

	return strings.Join(lines, "\n")
}
