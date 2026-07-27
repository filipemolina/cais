package components

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/constants"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

// AboutModalModel is the About overlay: the ASCII brand mark reserved for it
// (constants.LOGO), the version, the license, and the repo link. It is a
// read-only surface like the help overlay, and closes on the same three keys:
// the one that opened it (a toggle would be a fourth binding for one job), esc
// (the cancel every overlay answers), and q (the quitter's habit, which closes
// the overlay rather than quitting the app while it owns the keyboard).
type AboutModalModel struct{}

func (m AboutModalModel) Init() tea.Cmd { return nil }

func (m AboutModalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(keyMsg, keys.Global.About),
			key.Matches(keyMsg, keys.Overlay.Cancel),
			key.Matches(keyMsg, keys.Global.Quit):
			return m, cmds.CloseModal(nil)
		}
	}
	return m, nil
}

func (m AboutModalModel) View() tea.View {
	primary := lipgloss.NewStyle().Bold(true).Foreground(appstyles.Active.TextPrimary)
	accent := lipgloss.NewStyle().Bold(true).Foreground(appstyles.Active.Accent)
	dim := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim)
	muted := lipgloss.NewStyle().Foreground(appstyles.Active.TextMuted)

	label := func(s string) string { return muted.Render(s) }

	sections := []string{
		// The brand mark carries its own color (truecolor purple); render it
		// as-is so the embedded SGR codes land untouched.
		constants.LOGO,
		fmt.Sprintf("%s %s",
			primary.Render(constants.WORDMARK),
			dim.Render(constants.SLOGAN)),
		fmt.Sprintf("%s %s", label("version"), accent.Render(constants.Version())),
		fmt.Sprintf("%s %s   %s %s",
			label("license"), primary.Render("MIT"),
			label("repo"), primary.Render("github.com/filipemolina/stack-stitcher")),
	}

	hint := renderKeyHints([]KeyHint{
		hintAs(keys.Global.About, "close"),
		hintAs(keys.Overlay.Cancel, "close"),
		hintAs(keys.Global.Quit, "close"),
	}, appstyles.Active.TextMuted)
	sections = append(sections, hint)

	return tea.NewView(modalSurface(
		appstyles.Active.ModalBg,
		strings.Join(sections, "\n"),
	))
}

// AboutModal builds the About overlay.
func AboutModal() tea.Model {
	return AboutModalModel{}
}
