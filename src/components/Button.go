package components

import (
	"image/color"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
)

// ButtonSpec is one action control: the verb, the key that runs it, and the
// two facts that decide how it is painted. It is a struct rather than a
// parameter list because the last two are booleans, and `Button("Remove", "x",
// false, true)` says nothing at the call site about which is which.
type ButtonSpec struct {
	Text     string
	Shortcut string
	// Enabled is whether the key can be pressed right now. A disabled button
	// is dimmed rather than hidden: the row keeps its shape as focus moves, so
	// the panel's actions stay in one place instead of reflowing the panel body
	// every time Tab is pressed - and a dim control is the honest affordance for
	// "this needs the panel focused first".
	Enabled bool
	// Danger marks a destructive action, which is colored as one so the row
	// does not present "remove" as the peer of "restart".
	Danger bool
}

type ButtonModel struct {
	spec ButtonSpec
}

func (m ButtonModel) Init() tea.Cmd {
	return nil
}

func (m ButtonModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

// View renders the button as a recessed chip: a filled surface one row tall,
// no border.
//
// It used to be a rounded accent-outlined box three rows tall, which was the
// only element in the app built that way - everything else is either a chip
// (the title chip, the status pills) or a card recessed into its panel. An
// outline floating on the panel background reads as a control borrowed from
// another program, so the button now borrows the vocabulary it should have had:
// the pills' fill-and-padding shape, on the tier the empty-state cards already
// use for "inset into this panel".
//
// The chip carries its own surface rather than its parent's tier, which is the
// reverse of what it did as an outlined box. Focus is legible in the ink now -
// accent when live, dim when not - so the surface is free to stay put, and the
// recess deepens on its own when focus lifts the panel from tier 3 to tier 4.
func (m ButtonModel) View() tea.View {
	surface := appstyles.Active.BackgroundRecessed

	// The key is the accent and the label is plain text, matching how a
	// keystroke is set everywhere else it is offered - see renderEmptyCard's
	// hint line. A disabled chip drops to one flat tier so it reads as one dead
	// control rather than a live key beside a dead word.
	shortcut, label := appstyles.Active.Accent, appstyles.Active.TextPrimary
	switch {
	case !m.spec.Enabled:
		shortcut, label = appstyles.Active.TextDim, appstyles.Active.TextDim
	case m.spec.Danger:
		shortcut, label = appstyles.Active.StatusError, appstyles.Active.StatusError
	}

	// Every run carries the chip's surface. A run left unpainted here would
	// show the terminal's own background through the middle of the chip, which
	// is the bug docs/DESIGN.md's "Background tiers, and sealing them" is about.
	on := func(fg color.Color) lipgloss.Style {
		return lipgloss.NewStyle().Background(surface).Foreground(fg)
	}

	content := on(shortcut).Bold(true).Render(m.spec.Shortcut) + on(label).Render(" "+m.spec.Text)

	return tea.NewView(lipgloss.NewStyle().Background(surface).Padding(0, 1).Render(content))
}

func Button(spec ButtonSpec) tea.Model {
	return ButtonModel{spec: spec}
}
