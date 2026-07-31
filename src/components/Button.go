package components

import (
	"image/color"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
)

type ButtonModel struct {
	text     string
	shortcut string
	// bg is the background the button is drawn on. A button carries its
	// parent's tier rather than a tint of its own, so it stays flush with the
	// panel when focus lifts that panel from tier 3 to tier 4. Without an
	// explicit background the label would be the one run on the line with no
	// background set, showing the terminal's color through the button.
	bg color.Color
	// enabled is whether the key this button stands for can be pressed right
	// now. A disabled button is dimmed rather than hidden: the row keeps its
	// shape as focus moves, so the panel's actions stay in one place instead of
	// reflowing the panel body every time Tab is pressed - and a dim control is
	// the honest affordance for "this needs the panel focused first".
	enabled bool
}

func (m ButtonModel) Init() tea.Cmd {
	return nil
}

func (m ButtonModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m ButtonModel) View() tea.View {
	// The label color is set explicitly in both states. Left unset, an enabled
	// button's label would be the one run in the row with no foreground of its
	// own, which is the same class of bug as an unpainted background: it renders
	// in whatever color the terminal defaults to rather than the theme's.
	border, label := appstyles.Active.Accent, appstyles.Active.TextPrimary
	if !m.enabled {
		border, label = appstyles.Active.BorderDefault, appstyles.Active.TextDim
	}

	buttonStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		Padding(0, 1).
		BorderForeground(border).
		Foreground(label).
		Background(m.bg).
		BorderBackground(m.bg)

	return tea.NewView(buttonStyle.Render(m.shortcut + " " + m.text))
}

func Button(text string, shortcut string, bg color.Color, enabled bool) tea.Model {
	return ButtonModel{
		text:     text,
		shortcut: shortcut,
		bg:       bg,
		enabled:  enabled,
	}
}
