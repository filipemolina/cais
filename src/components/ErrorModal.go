package components

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

// ErrorModalModel is a modal that displays an error message and closes on
// esc or any dismiss key. It is used for foreground errors (docker actions,
// config loads) where a modal is less disruptive than the banner.
type ErrorModalModel struct {
	message string
	width   int
}

func (m ErrorModalModel) Init() tea.Cmd { return nil }

func (m ErrorModalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch {
	case key.Matches(keyMsg, keys.Global.Back),
		key.Matches(keyMsg, keys.Overlay.Cancel),
		key.Matches(keyMsg, keys.Overlay.Yes),
		key.Matches(keyMsg, keys.Overlay.No):
		return m, cmds.CloseModal(nil)
	}

	return m, nil
}

func (m ErrorModalModel) View() tea.View {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(appstyles.Active.Danger).
		Background(appstyles.Active.ModalBg).
		MarginBottom(1)

	messageStyle := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextPrimary).
		Background(appstyles.Active.ModalBg).
		Width(m.width)

	hintStyle := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextDim).
		Background(appstyles.Active.ModalBg).
		MarginTop(1)

	content := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Error"),
		messageStyle.Render(m.message),
		hintStyle.Render("Press esc to dismiss"),
	)

	return tea.NewView(modalSurface(appstyles.Active.ModalBg, content))
}

// ErrorModal creates a new error modal with the given message.
func ErrorModal(message string, terminalWidth int) tea.Model {
	// Constrain the message width to half the terminal or 60, whichever is smaller.
	width := min(60, terminalWidth/2)
	if width < 20 {
		width = 20
	}

	return ErrorModalModel{
		message: message,
		width:   width,
	}
}
