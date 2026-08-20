package helpoverlay

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/keys"
	"github.com/sahilm/fuzzy"
)

// contentWidth is the column the overlay's hints wrap to: the terminal minus
// the modal chrome and a margin, capped.
func (m Model) contentWidth() int {
	return max(24, min(helpOverlayMaxWidth, m.termWidth-16))
}

// renderScope renders one scope as a title line over one line per entry —
// each key/description pair gets its own row rather than being packed
// side-by-side and wrapped, which is what made a scope with several keys
// read as a run-on paragraph instead of a list. Unavailable rows are dimmed
// whole; available rows get the footer's treatment (key bold, description
// muted). Keys are padded to the widest key in the scope so the descriptions
// line up in a column.
func renderScope(scope keys.Scope, width int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(appstyles.Active.Accent)
	dimStyle := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextDim)
	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(appstyles.Active.TextPrimary)
	descStyle := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextMuted)

	keyWidth := 0
	for _, entry := range scope.Entries {
		keyWidth = max(keyWidth, lipgloss.Width(entry.Binding.Help().Key))
	}

	lines := []string{titleStyle.Render(scope.Title)}
	for _, entry := range scope.Entries {
		help := entry.Binding.Help()
		key := lipgloss.NewStyle().Width(keyWidth).Render(help.Key)

		var row string
		if entry.Available {
			row = keyStyle.Render(key) + descStyle.Render("  "+help.Desc)
		} else {
			row = dimStyle.Render(key + "  " + help.Desc)
		}
		lines = append(lines, lipgloss.NewStyle().Width(width).Render(row))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// filteredScope narrows scope to the entries whose key or description
// fuzzy-matches query, in their original order. ok is false when nothing in
// the scope matched, so the caller can drop the scope (and its title)
// entirely rather than show a heading over nothing.
func filteredScope(scope keys.Scope, query string) (result keys.Scope, ok bool) {
	if query == "" {
		return scope, true
	}

	texts := make([]string, len(scope.Entries))
	for i, entry := range scope.Entries {
		help := entry.Binding.Help()
		texts[i] = help.Key + " " + help.Desc
	}

	matches := fuzzy.Find(query, texts)
	if len(matches) == 0 {
		return keys.Scope{}, false
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].Index < matches[j].Index })

	original := scope.Entries
	scope.Entries = make([]keys.Entry, len(matches))
	for i, match := range matches {
		scope.Entries[i] = original[match.Index]
	}
	return scope, true
}

// filteredCatalog is the catalog narrowed by the current filter query, scope
// by scope. An empty query returns the catalog unchanged.
func (m Model) filteredCatalog() []keys.Scope {
	query := strings.TrimSpace(m.filterQuery)
	out := make([]keys.Scope, 0, len(m.catalog))
	for _, scope := range m.catalog {
		if filtered, ok := filteredScope(scope, query); ok {
			out = append(out, filtered)
		}
	}
	return out
}

// renderComposeFiles names the candidates that lost to the loaded file, in
// the priority order Docker resolves them. It renders nothing when the
// winner was the only candidate - the footer already says its name.
func renderComposeFiles(files []string) string {
	if len(files) <= 1 {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(appstyles.Active.Accent)
	noteStyle := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextDim)
	activeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(appstyles.Active.TextPrimary)

	lines := []string{
		titleStyle.Render("Compose files"),
		noteStyle.Render("docker uses the first of these that exists:"),
		activeStyle.Render(files[0]) + noteStyle.Render("  (in use)"),
	}
	for _, name := range files[1:] {
		lines = append(lines, noteStyle.Render(name))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// contentLines renders every scope in the filtered catalog plus the compose
// files section, and flattens the result to one list of already-wrapped
// lines, with a blank line between sections — the section separation the
// catalog has always had, preserved across filtering. Flattening before
// windowing is what lets a section taller than the window still be read: the
// window cuts between lines, not between sections. The compose files section
// is not itself filterable — it is not a keybinding — so it only appears
// with an empty query.
func (m Model) contentLines() []string {
	width := m.contentWidth()
	catalog := m.filteredCatalog()
	query := strings.TrimSpace(m.filterQuery)

	if len(catalog) == 0 && query != "" {
		return []string{lipgloss.NewStyle().
			Foreground(appstyles.Active.TextDim).
			Width(width).
			Render(fmt.Sprintf("No shortcuts match %q.", query))}
	}

	var lines []string
	for i, scope := range catalog {
		if i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, strings.Split(renderScope(scope, width), "\n")...)
	}

	if query == "" {
		if files := renderComposeFiles(m.composeFiles); files != "" {
			if len(lines) > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, strings.Split(files, "\n")...)
		}
	}
	return lines
}

// maxOffset is the furthest the content can scroll: enough to bring the last
// line into view, never past it.
func (m Model) maxOffset() int {
	return max(0, len(m.contentLines())-m.contentRows())
}

// contentRows is how many lines of content fit on this terminal: the
// terminal minus everything the overlay spends on chrome — the modal border
// and padding, the title, the filter bar (when active), the overflow counts
// and the hint line, plus the blank line between each.
//
// It is MEASURED rather than counted from a constant, by assembling the
// overlay around a single content line and subtracting it, the same way a
// hardcoded number would be right at one width and wrong at the next.
func (m Model) contentRows() int {
	chrome := lipgloss.Height(m.assemble("x", "999 above · 999 below")) - 1
	return max(1, m.termHeight-chrome)
}

// renderFilterBar renders the `/`-filter row: "/" plus the live query (or the
// focused input while typing) plus a hint. It renders empty when no filter is
// active, so assemble can splice it in unconditionally and contentRows —
// which measures assemble at the model's current state — picks up its height
// automatically.
func (m Model) renderFilterBar() string {
	if !m.filterTyping && !m.filterApplied {
		return ""
	}

	slash := lipgloss.NewStyle().Foreground(appstyles.Active.TextPrimary).Bold(true).Render("/")
	hint := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).Render("esc to clear")

	if m.filterTyping {
		return slash + " " + m.filterInput.View() + "  " + hint
	}
	query := lipgloss.NewStyle().Foreground(appstyles.Active.TextMuted).Render(m.filterQuery)
	return slash + " " + query + "  " + hint
}

// overflowLabel reports how many content lines are hidden above and below the
// current window.
func (m Model) overflowLabel() string {
	lines := m.contentLines()
	offset := min(m.offset, m.maxOffset())
	end := min(offset+m.contentRows(), len(lines))

	var parts []string
	if offset > 0 {
		parts = append(parts, fmt.Sprintf("%d above", offset))
	}
	if below := len(lines) - end; below > 0 {
		parts = append(parts, fmt.Sprintf("%d below", below))
	}
	return strings.Join(parts, " · ")
}

// assemble wraps a body of content lines in the overlay's chrome: the title
// above, the filter bar when active, then the overflow counts and the hint
// line below. contentRows measures itself against this, so the two can never
// disagree about what the chrome costs.
func (m Model) assemble(body, overflow string) string {
	windowed := overflow != ""
	width := m.contentWidth()
	dim := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).Width(width)

	sections := []string{chrome.ModalTitle("Keyboard shortcuts")}
	if bar := m.renderFilterBar(); bar != "" {
		sections = append(sections, bar)
	}
	sections = append(sections, body)

	if windowed {
		sections = append(sections, dim.Render(overflow))
	}

	// The overlay's own keys, built from the same bindings as everything
	// else: it owns the keyboard while open, and an overlay advertises its
	// own keys because the footer is hidden beneath it. Scrolling and
	// closing are only advertised while the filter input is not eating
	// every keystroke — its own bar already explains esc while typing, and ?
	// types a literal character rather than closing.
	var hints []chrome.KeyHint
	if m.filterTyping {
		hints = append(hints, chrome.HintAs(keys.Overlay.Submit, "apply"))
	} else {
		if windowed {
			hints = append(hints, chrome.HintAs(keys.Overlay.Navigation, "scroll"))
		}
		hints = append(hints, chrome.HintAs(keys.List.Filter, "filter"))
		hints = append(hints,
			chrome.HintAs(keys.Global.Help, "close"),
			chrome.HintAs(keys.Overlay.Cancel, "close"),
			chrome.HintAs(keys.Global.Quit, "close"),
		)
	}
	sections = append(sections, chrome.RenderKeyHints(hints, appstyles.Active.TextMuted))

	return chrome.ModalSurface(appstyles.Active.ModalBg, strings.Join(sections, "\n\n"))
}

func (m Model) View() tea.View {
	m.filterInput.SetWidth(max(0, m.contentWidth()-6))

	lines := m.contentLines()
	windowed := m.maxOffset() > 0
	offset := min(m.offset, m.maxOffset())
	end := min(offset+m.contentRows(), len(lines))

	overflow := ""
	if windowed {
		overflow = m.overflowLabel()
	}
	return tea.NewView(m.assemble(strings.Join(lines[offset:end], "\n"), overflow))
}
