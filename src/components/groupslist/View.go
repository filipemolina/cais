package groupslist

import (
	"fmt"
	"image/color"
	"io"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/chrome"
)

/*
 * Styling by creating a custom delegate
 */

type GroupsListCustomDelegate struct {
	isParentFocused bool
	activeIndex     int
}

func (d GroupsListCustomDelegate) Height() int                             { return 4 }
func (d GroupsListCustomDelegate) Spacing() int                            { return 0 }
func (d GroupsListCustomDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

// Render handles the actual drawing of the item
func (d GroupsListCustomDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	// Cast the generic list.Item back to our specific GroupListItem
	item, ok := listItem.(apptypes.GroupListItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()
	isActive := index == d.activeIndex
	var titleColor color.Color

	if isActive {
		titleColor = appstyles.Active.TextPrimary
	} else {
		titleColor = appstyles.Active.TextMuted
	}

	rowBg := chrome.ListRowBg(isActive, d.isParentFocused)

	// The row's left edge is the same solid bar the nav uses for its active
	// tab ("▌"), so list rows and the nav agree on thickness. State is carried
	// by color alone: accent = cursor row, primary = selected, muted = default.
	barColor := appstyles.Active.TextMuted

	if isActive {
		barColor = appstyles.Active.Accent
	} else if isSelected && d.isParentFocused {
		barColor = appstyles.Active.TextPrimary
	}

	wrapperStyle := lipgloss.NewStyle().
		Width(m.Width() - 1).
		Padding(1).
		Background(rowBg)

	// The title style only bolds the active row; the selected row's bold comes
	// from the wrapper, which is why it is applied here rather than in the title.
	if isSelected && d.isParentFocused && !isActive {
		wrapperStyle = wrapperStyle.Bold(true)
	}

	titleStyle := lipgloss.NewStyle().
		Bold(isActive).
		Foreground(titleColor).
		Background(rowBg).
		Width(m.Width() - 2)

	title := titleStyle.Render(item.Title())

	// The status dot rides the far right of the title line: a green full
	// circle when every member service is running, an amber half circle when
	// some but not all are, and nothing when the group is stopped. It uses
	// the same glyphs and status colors as the member table's dot column, so
	// the two read as one visual language.
	dot := statusDot(item, rowBg)

	// The title yields the dot's column (Width above is one less than the
	// row), so joining the dot after it lands flush at the right edge without
	// overflowing the row. A stopped group's empty dot adds nothing, keeping
	// the row the same width either way.
	titleRow := lipgloss.JoinHorizontal(lipgloss.Left, title, dot)

	content := wrapperStyle.Render(lipgloss.JoinVertical(lipgloss.Left, titleRow))

	// The bar spans the row's full height, one ▌ per line, rather than a sliver
	// at the top - the nav's single-line bar stretched to the row's height.
	bar := chrome.BarColumn(barColor, rowBg, content)

	// Seal the row against its own background before handing it to the list, so
	// the active row keeps its lighter surface color instead of being flattened
	// to the panel's when the list is sealed - see appstyles.FillBackground.
	row := appstyles.FillBackground(rowBg, lipgloss.JoinHorizontal(lipgloss.Left, bar, content))

	// Print the styled string to the Bubble Tea io.Writer
	fmt.Fprint(w, row)
}

// statusDot returns the styled status glyph for a group row: a green full
// circle when every member service is running, an amber half circle when some
// but not all are, and an empty string when the group is stopped (no services
// running). It mirrors the member table's dot column - same glyphs, same
// status colors - so the group list and the details panel read as one visual
// language. The dot is rendered on the row background so it stays legible
// across selection/focus states.
func statusDot(item apptypes.GroupListItem, rowBg color.Color) string {
	var glyph string
	var dotColor color.Color

	switch {
	case item.Total > 0 && item.Running == item.Total:
		glyph, dotColor = "●", appstyles.Active.StatusRunning
	case item.Running > 0:
		glyph, dotColor = "◐", appstyles.Active.StatusStarting
	default:
		return ""
	}

	return lipgloss.NewStyle().
		Foreground(dotColor).
		Background(rowBg).
		Render(glyph)
}

// statsLine is the counts footer, in the longest form that fits `width`
// columns. It has to fit on one row: it is the panel's last line, so wrapping
// it eats into the padding below instead of just pushing the list down.
//
// The ungrouped count only means something once groups exist to be left out
// of - with none yet, every service is ungrouped by definition (see
// groupdetailspanel.renderServiceOverview), so the number would just repeat
// Services rather than warn about anything. It sheds before the core three
// on a narrow terminal (the "shed whole things" rule in docs/DESIGN.md,
// *Narrow terminals*): the standing count is a courtesy, not the panel's own
// verb, so it goes first when there is no room for it.
func statsLine(stats cmds.SetHomeStatsMsg, width int) string {
	core := fmt.Sprintf("%d %s · %d %s · %d running",
		stats.Groups, plural(stats.Groups, "group"),
		stats.Services, plural(stats.Services, "service"),
		stats.Running)
	shortCore := fmt.Sprintf("%d grp · %d svc · %d run", stats.Groups, stats.Services, stats.Running)

	if stats.Groups > 0 && stats.Ungrouped > 0 {
		if withNote := core + fmt.Sprintf(" · %d ungrouped, always run", stats.Ungrouped); lipgloss.Width(withNote) <= width {
			return withNote
		}
		if withNote := shortCore + fmt.Sprintf(" · %d ungrp", stats.Ungrouped); lipgloss.Width(withNote) <= width {
			return withNote
		}
	}

	if lipgloss.Width(core) <= width {
		return core
	}

	return shortCore
}

// plural is the naive English plural of word for n: enough for the handful of
// countable nouns the UI puts in front of a number.
func plural(n int, word string) string {
	if n == 1 {
		return word
	}

	return word + "s"
}

func (m Model) View() tea.View {
	// 3-tier background system: tier 3 (panel) when unfocused,
	// tier 4 (elevated) when focused. The focus state is shown by the
	// background lifting, not by a border.
	bg := chrome.PanelBg(m.isFocused)

	// The panel fills exactly the box AppModel handed it, so the tier-3
	// background covers the full body region and both panels are the same
	// height regardless of how much content they hold.
	wrapper := chrome.FitBox(chrome.ListWrapperStyle.Background(bg), m.panelWidth, m.panelHeight)

	// Rows left for the sections below, inside the wrapper padding.
	frameW, frameH := chrome.ListWrapperStyle.GetFrameSize()
	contentWidth := max(0, m.panelWidth-frameW)
	contentHeight := max(0, m.panelHeight-frameH)

	var sections []string

	if len(m.list.Items()) == 0 {
		// Render the list title even when empty, using the same accent-chip
		// style as the right panel's Details title (chrome.PanelFrame). The
		// MarginLeft(2) matches the gutter the bubbles list TitleBar adds, so
		// the empty state's chip lines up with the non-empty one's.
		titleRow := appstyles.NormalTitle().MarginLeft(2).Render(m.list.Title)
		emptyStyle := lipgloss.NewStyle().
			Foreground(appstyles.Active.TextMuted).
			Background(bg).
			Padding(2, 2)
		// Width-constrained so the hint wraps inside the panel instead of
		// widening it past its box.
		emptyContent := chrome.FitBox(emptyStyle, contentWidth, max(0, contentHeight-m.footerHeight())).Render(
			"No groups yet.\nPress n to create one, or add profiles to services in your compose file.",
		)
		sections = append(sections, appstyles.FillBackground(bg, lipgloss.JoinVertical(lipgloss.Left, titleRow, emptyContent)))
	} else {
		// The title chip is restyled here, on a copy, rather than in the
		// constructor - see appstyles.NormalTitle for why.
		l := m.list
		l.Styles.Title = appstyles.NormalTitle()
		sections = append(sections, l.View())
	}

	// The stats sit on the panel's last row rather than above the list title:
	// as a header they crowded the title chip, and they read as a summary of
	// what is above them anyway. The list fills the height it was given, so
	// appending the line here pins it to the bottom of the panel.
	if m.hasStats {
		footerStyle := lipgloss.NewStyle().
			Foreground(appstyles.Active.TextDim).
			Background(bg).
			Padding(0, 1)

		frameW, _ := footerStyle.GetFrameSize()
		sections = append(sections, chrome.FitBox(footerStyle, contentWidth, 0).Render(statsLine(m.stats, contentWidth-frameW)))
	}

	// JoinVertical pads the shorter of the stats footer / list out to the
	// widest with unstyled spaces, so seal the joined block against the panel
	// tier. Rows arrive already sealed against their own background.
	content := appstyles.FillBackground(bg, lipgloss.JoinVertical(lipgloss.Left, sections...))

	v := tea.NewView(wrapper.Render(content))
	return v
}
