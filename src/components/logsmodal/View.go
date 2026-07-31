package logsmodal

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

// logsModalWrapper builds the near-full-screen overlay's chrome fresh each
// call, so it re-reads appstyles.Active instead of freezing whichever theme
// was active when the package loaded.
//
// BorderBackground matters as much as Background here: without it lipgloss
// leaves the border cells on the terminal's default color, outlining a
// near-full-screen overlay in the wrong shade.
func logsModalWrapper() lipgloss.Style {
	return lipgloss.NewStyle().
		Padding(0, 1).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(appstyles.Active.Accent).
		BorderBackground(appstyles.Active.PanelBg).
		Background(appstyles.Active.PanelBg)
}

func (m Model) View() tea.View {
	title := chrome.ModalTitle("logs: " + m.title)

	followState := "off"
	if m.follow {
		followState = "on"
	}

	// Built from the bindings rather than written out, so rebinding follow or
	// cancel cannot leave this line advertising the old key.
	footer := chrome.RenderKeyHints([]chrome.KeyHint{
		chrome.HintAs(keys.List.Navigate, "scroll"),
		chrome.HintAs(keys.Overlay.Follow, fmt.Sprintf("follow (%s)", followState)),
		chrome.HintAs(keys.Overlay.Cancel, "quit"),
	}, appstyles.Active.TextMuted)

	if m.ended {
		footer = lipgloss.NewStyle().
			Foreground(appstyles.Active.TextMuted).
			Render("stream ended · ") + footer
	}

	body := m.viewport.View()
	if m.err != nil {
		body = lipgloss.NewStyle().
			Foreground(appstyles.Active.TextPrimary).
			Render("Error: " + m.err.Error())
	}

	// The title and footer are far shorter than the viewport, so JoinVertical
	// pads them out with unstyled spaces; seal them against the modal's
	// background before the wrapper draws its border.
	content := appstyles.FillBackground(
		appstyles.Active.PanelBg,
		lipgloss.JoinVertical(lipgloss.Left, title, body, footer),
	)

	return tea.NewView(logsModalWrapper().Render(content))
}
