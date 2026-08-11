package healthcheckpickermodal

import (
	"fmt"
	"io"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/keys"
)

type templateDelegate struct{}

func (d templateDelegate) Height() int                             { return 1 }
func (d templateDelegate) Spacing() int                            { return 0 }
func (d templateDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d templateDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(templateItem)
	if !ok {
		return
	}

	style := lipgloss.NewStyle().Foreground(appstyles.Active.TextMuted)
	if index == m.Index() {
		style = style.Foreground(appstyles.Active.TextPrimary).Bold(true)
	}

	fmt.Fprint(w, style.Render(item.template.Name))
}

func (m Model) View() tea.View {
	title := fmt.Sprintf("Add healthcheck to %q", m.serviceName)
	if m.replacing {
		title = fmt.Sprintf("Replace healthcheck on %q", m.serviceName)
	}

	sections := []string{chrome.ModalTitle(title), m.list.View()}

	if t, ok := m.selectedTemplate(); ok && t.Generic {
		sections = append(sections,
			lipgloss.NewStyle().Foreground(appstyles.Active.TextMuted).Render("Port inside the container:"),
			m.portInput.View(),
		)
	}

	if m.errMsg != "" {
		sections = append(sections, lipgloss.NewStyle().Foreground(appstyles.Active.Danger).Render(m.errMsg))
	}

	submitDesc := "add"
	if m.replacing {
		submitDesc = "replace"
	}
	sections = append(sections, "", chrome.ModalHints(
		chrome.HintFor(keys.List.Navigate),
		chrome.HintAs(keys.Overlay.Submit, submitDesc),
		chrome.HintFor(keys.Overlay.Cancel),
	))

	return tea.NewView(chrome.ModalSurface(
		appstyles.Active.ModalBg,
		lipgloss.JoinVertical(lipgloss.Left, sections...),
	))
}
