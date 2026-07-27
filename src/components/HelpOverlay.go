package components

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

// helpOverlayMaxWidth caps the content column so hint runs wrap in a few
// places on wide terminals rather than stretching into one unreadable line.
const helpOverlayMaxWidth = 64

// HelpOverlayModel is the ? overlay: every key in the app, grouped by scope
// and rendered from keys.Catalog, so what it says is what the handlers do.
// Rows the user could not press in the screen it was opened from are dimmed.
// It also names the compose-file candidates that lost to the loaded one - the
// footer only has room to count them.
type HelpOverlayModel struct {
	catalog      []keys.Scope
	composeFiles []string
	termWidth    int
}

func (m HelpOverlayModel) Init() tea.Cmd { return nil }

func (m HelpOverlayModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width

	case tea.KeyPressMsg:
		// Any of the three closes: ? is the toggle that opened it, esc is
		// the cancel every overlay answers, q is the quitter's habit. Only
		// the overlay closes - q never quits the app from here, because the
		// overlay owns the keyboard while it is open.
		switch {
		case key.Matches(msg, keys.Global.Help),
			key.Matches(msg, keys.Overlay.Cancel),
			key.Matches(msg, keys.Global.Quit):
			return m, cmds.CloseModal(nil)
		}
	}

	return m, nil
}

// contentWidth is the column the overlay's hints wrap to: the terminal minus
// the modal chrome and a margin, capped.
func (m HelpOverlayModel) contentWidth() int {
	return max(24, min(helpOverlayMaxWidth, m.termWidth-16))
}

// renderScope renders one scope as a title line over its hint runs, wrapped
// to width. Unavailable rows are dimmed whole; available rows get the
// footer's treatment (key bold, description muted).
func renderScope(scope keys.Scope, width int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(appstyles.Accent)
	dimStyle := lipgloss.NewStyle().
		Foreground(appstyles.TextDim)
	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(appstyles.TextPrimary)
	descStyle := lipgloss.NewStyle().
		Foreground(appstyles.TextMuted)

	runs := make([]string, 0, len(scope.Entries)*2)
	for i, entry := range scope.Entries {
		if i > 0 {
			runs = append(runs, dimStyle.Render(" · "))
		}

		help := entry.Binding.Help()
		if entry.Available {
			runs = append(runs, keyStyle.Render(help.Key)+descStyle.Render(" "+help.Desc))
		} else {
			runs = append(runs, dimStyle.Render(help.Key+" "+help.Desc))
		}
	}

	body := lipgloss.NewStyle().
		Width(width).
		Render(lipgloss.JoinHorizontal(lipgloss.Left, runs...))

	return lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render(scope.Title), body)
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
		Foreground(appstyles.Accent)
	noteStyle := lipgloss.NewStyle().
		Foreground(appstyles.TextDim)
	activeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(appstyles.TextPrimary)

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

func (m HelpOverlayModel) View() tea.View {
	width := m.contentWidth()

	sections := []string{
		lipgloss.NewStyle().
			Bold(true).
			Foreground(appstyles.TextPrimary).
			Render("Keyboard shortcuts"),
	}

	for _, scope := range m.catalog {
		sections = append(sections, renderScope(scope, width))
	}

	if files := renderComposeFiles(m.composeFiles); files != "" {
		sections = append(sections, files)
	}

	// The overlay's own keys, built from the same bindings as everything
	// else: it owns the keyboard while open, and an overlay advertises its
	// own keys because the footer is hidden beneath it.
	hint := renderKeyHints([]KeyHint{
		hintAs(keys.Global.Help, "close"),
		hintAs(keys.Overlay.Cancel, "close"),
		hintAs(keys.Global.Quit, "close"),
	}, appstyles.TextMuted)
	sections = append(sections, hint)

	return tea.NewView(modalSurface(
		appstyles.PanelBackgroundColor,
		strings.Join(sections, "\n\n"),
	))
}

// HelpOverlay builds the help overlay for the screen described by ctx (which
// keys are pressable), the compose-file candidates in priority order, and the
// terminal width for wrapping.
func HelpOverlay(ctx keys.Context, composeFiles []string, termWidth int) tea.Model {
	return HelpOverlayModel{
		catalog:      keys.Catalog(ctx),
		composeFiles: composeFiles,
		termWidth:    termWidth,
	}
}

