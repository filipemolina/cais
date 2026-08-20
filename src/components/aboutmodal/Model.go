package aboutmodal

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/banner"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/constants"
	"github.com/filipemolina/cais/src/keys"
)

// Model is the About overlay: the figlet brand mark (banner.Banner), the
// version, the license, and the repo link. It is a read-only surface like
// the help overlay, and closes on the same three keys:
// the one that opened it (a toggle would be a fourth binding for one job), esc
// (the cancel every overlay answers), and q (the quitter's habit, which closes
// the overlay rather than quitting the app while it owns the keyboard).
type Model struct{}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m Model) View() tea.View {
	primary := lipgloss.NewStyle().Bold(true).Foreground(appstyles.Active.TextPrimary)
	accent := lipgloss.NewStyle().Bold(true).Foreground(appstyles.Active.Accent)
	dim := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim)
	muted := lipgloss.NewStyle().Foreground(appstyles.Active.TextMuted)

	label := func(s string) string { return muted.Render(s) }

	sections := []string{
		chrome.ModalTitle("About"),
		// The figlet carries its own gradient, rendered fresh from the active
		// theme on every call; render it as-is so the embedded SGR codes land
		// untouched.
		banner.Banner(appstyles.Active),
		fmt.Sprintf("%s %s",
			primary.Render(constants.WORDMARK),
			dim.Render(constants.SLOGAN)),
		fmt.Sprintf("%s %s", label("version"), accent.Render(constants.Version())),
		fmt.Sprintf("%s %s   %s %s",
			label("license"), primary.Render("MIT"),
			label("repo"), primary.Render("github.com/filipemolina/cais")),
	}

	hint := chrome.RenderKeyHints([]chrome.KeyHint{
		chrome.HintAs(keys.Global.About, "close"),
		chrome.HintAs(keys.Overlay.Cancel, "close"),
		chrome.HintAs(keys.Global.Quit, "close"),
	}, appstyles.Active.TextMuted)
	sections = append(sections, hint)

	return tea.NewView(chrome.ModalSurface(
		appstyles.Active.ModalBg,
		strings.Join(sections, "\n"),
	))
}

// New builds the About overlay.
func New() tea.Model {
	return Model{}
}
