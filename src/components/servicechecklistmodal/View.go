package servicechecklistmodal

import (
	"fmt"
	"io"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

type serviceChecklistDelegate struct{}

func (d serviceChecklistDelegate) Height() int                             { return 1 }
func (d serviceChecklistDelegate) Spacing() int                            { return 0 }
func (d serviceChecklistDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d serviceChecklistDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(apptypes.CheckableServiceItem)
	if !ok {
		return
	}

	style := lipgloss.NewStyle().Foreground(appstyles.Active.TextMuted)
	if index == m.Index() {
		style = style.Foreground(appstyles.Active.TextPrimary).Bold(true)
	}

	fmt.Fprint(w, style.Render(item.Title()))
}

// checklistHints is the modal's own help line. The footer bar is hidden
// behind the modal while this is open, so the keys it takes over - space,
// enter, esc - have to be advertised here or nowhere. Two lines rather than
// one so the modal stays as narrow as its list.
//
// submitDesc names what Enter confirms in this mode: "create group" for a
// new group, "save changes" for an edit. Enter is "confirm" everywhere;
// here what it confirms is worth naming, since it is the step that writes
// to the compose file.
//
// TextMuted, not the bar's TextDim: this sits on the modal's light surface,
// where TextDim barely separates from the background.
func checklistHints(submitDesc string) string {
	return lipgloss.JoinVertical(lipgloss.Left,
		chrome.RenderKeyHints([]chrome.KeyHint{
			chrome.HintFor(keys.List.Navigate),
			chrome.HintFor(keys.Overlay.Toggle),
		}, appstyles.Active.TextMuted),
		chrome.RenderKeyHints([]chrome.KeyHint{
			chrome.HintAs(keys.Overlay.Submit, submitDesc),
			chrome.HintFor(keys.Overlay.Cancel),
		}, appstyles.Active.TextMuted),
	)
}

func (m Model) View() tea.View {
	submitDesc := "create group"
	if m.isEdit {
		submitDesc = "save changes"
	}

	title := fmt.Sprintf("Select services for %q", m.groupName)
	if m.isEdit {
		title = fmt.Sprintf("Edit members of %q", m.groupName)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, chrome.ModalTitle(title), m.list.View(), "", checklistHints(submitDesc))

	return tea.NewView(chrome.ModalSurface(appstyles.Active.ModalBg, content))
}
