package composefilepickermodal

import (
	"fmt"
	"io"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/keys"
)

type composeFilePickerDelegate struct{}

func (d composeFilePickerDelegate) Height() int                             { return 1 }
func (d composeFilePickerDelegate) Spacing() int                            { return 0 }
func (d composeFilePickerDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d composeFilePickerDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(apptypes.ComposeFileItem)
	if !ok {
		return
	}

	style := lipgloss.NewStyle().Foreground(appstyles.Active.TextMuted)
	if index == m.Index() {
		style = style.Foreground(appstyles.Active.TextPrimary).Bold(true)
	} else if item.Active {
		// The file already loaded is worth marking even off the cursor, so
		// "switch to the one I'm on" reads as a no-op rather than a choice.
		style = style.Foreground(appstyles.Active.Accent)
	}

	fmt.Fprint(w, style.Render(item.Title()))
}

// pickerHints is the modal's own help line; the footer is hidden behind the
// modal, so the keys it takes over are advertised here. Two lines, like the
// checklist modal's, so the modal stays as narrow as its list.
func pickerHints() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		chrome.RenderKeyHints([]chrome.KeyHint{
			chrome.HintFor(keys.List.Navigate),
		}, appstyles.Active.TextMuted),
		chrome.RenderKeyHints([]chrome.KeyHint{
			chrome.HintAs(keys.Overlay.Submit, "switch file"),
			chrome.HintFor(keys.Overlay.Cancel),
		}, appstyles.Active.TextMuted),
	)
}

func (m Model) View() tea.View {
	content := lipgloss.JoinVertical(lipgloss.Left, chrome.ModalTitle("Switch compose file"), m.list.View(), "", pickerHints())

	return tea.NewView(chrome.ModalSurface(appstyles.Active.ModalBg, content))
}
