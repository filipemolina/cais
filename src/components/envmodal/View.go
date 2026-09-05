package envmodal

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/keys"
)

// maskWidth is the fixed number of dots a hidden value renders as, so the
// masked column reveals neither the value nor its length.
const maskWidth = 8

// modalWidth is the column width the table renders at inside the modal. The
// modal surface adds its own padding around this.
const modalWidth = 60

func (m Model) View() tea.View {
	var body string

	switch {
	case m.loading:
		body = chrome.ModalHints(
			chrome.HintFor(keys.Overlay.Cancel),
		)
		content := lipgloss.JoinVertical(lipgloss.Left,
			chrome.ModalTitle("Env"),
			"Loading .env…",
			"",
			body,
		)
		return tea.NewView(chrome.ModalSurface(appstyles.Active.ModalBg, content))

	case m.loadErr != nil:
		content := lipgloss.JoinVertical(lipgloss.Left,
			chrome.ModalTitle("Env"),
			lipgloss.NewStyle().Foreground(appstyles.Active.Danger).Render(m.loadErr.Error()),
			"",
			chrome.ModalHints(chrome.HintFor(keys.Overlay.Cancel)),
		)
		return tea.NewView(chrome.ModalSurface(appstyles.Active.ModalBg, content))

	case len(m.list.Items()) == 0:
		content := lipgloss.JoinVertical(lipgloss.Left,
			chrome.ModalTitle("Env"),
			"This .env file has no variables yet.",
			"",
			chrome.ModalHints(
				chrome.HintFor(keys.List.New),
				chrome.HintFor(keys.Overlay.Cancel),
			),
		)
		return tea.NewView(chrome.ModalSurface(appstyles.Active.ModalBg, content))

	default:
		body = m.renderTable(modalWidth)
	}

	hints := chrome.ModalHints(
		chrome.HintFor(keys.List.New),
		chrome.HintFor(keys.List.Edit),
		chrome.HintFor(keys.List.Delete),
		chrome.HintFor(keys.Env.Reveal),
		chrome.HintFor(keys.Env.Copy),
		chrome.HintFor(keys.Details.EditFile),
		chrome.HintFor(keys.Env.RawEdit),
		chrome.HintFor(keys.Overlay.Cancel),
	)

	var header string
	if m.parseErrorCount > 0 {
		header = lipgloss.NewStyle().
			Foreground(appstyles.Active.Danger).
			Render(fmt.Sprintf("%d parse error%s", m.parseErrorCount, plural(m.parseErrorCount)))
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		chrome.ModalTitle("Env"),
		header,
		body,
		"",
		hints,
	)

	return tea.NewView(chrome.ModalSurface(appstyles.Active.ModalBg, content))
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// keyColWidth splits the row into a key column and a value column, capping
// the key at 28 columns so long keys don't starve the value.
func keyColWidth(contentWidth int) (int, int) {
	keyWidth := min(28, contentWidth/2)
	keyWidth = max(keyWidth, 8)
	valWidth := max(1, contentWidth-keyWidth-1) // -1 for the gap between columns
	return keyWidth, valWidth
}

// renderTable renders the KEY / VALUE table for the loaded entries.
//
// The rows come from the list rather than a loop over every entry, which is
// what keeps a long .env inside the modal: the loop drew all of them and the
// surplus ran off the bottom of the screen.
func (m Model) renderTable(contentWidth int) string {
	// m.list is a value, so this delegate swap is local to the frame being
	// rendered and cannot leak the reveal into the model.
	l := m.list
	l.SetDelegate(envDelegate{revealed: m.revealed})

	return lipgloss.JoinVertical(lipgloss.Left,
		m.renderHeader(contentWidth),
		chrome.PanelRule(contentWidth),
		l.View(),
	)
}

func (m Model) renderHeader(contentWidth int) string {
	dim := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).Bold(true)

	return lipgloss.JoinHorizontal(lipgloss.Left,
		lipgloss.NewStyle().Width(1).Render(""),
		dim.Width(28+1).Render("KEY"),
		dim.Width(max(1, contentWidth-29)).Render("VALUE"),
	)
}

// envDelegate renders the rows. It is a struct rather than a closure because
// bubbles calls Render per visible row and needs Height and Spacing first.
//
// revealed is set on a fresh delegate each frame rather than stored once. The
// list is built before anything is revealed, so a delegate that kept whatever
// it was constructed with would have revealed row 0 for the life of the modal -
// the zero value of an index is a real row.
type envDelegate struct {
	revealed int
}

func (d envDelegate) Height() int  { return 1 }
func (d envDelegate) Spacing() int { return 0 }

func (d envDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d envDelegate) Render(w io.Writer, l list.Model, index int, listItem list.Item) {
	item, ok := listItem.(envItem)
	if !ok {
		return
	}

	fmt.Fprint(w, renderEnvRow(item.EnvEntry, l.Width(), index == l.Index(), index == d.revealed))
}

// renderEnvRow renders one entry. Variable rows are a two-column key/value row;
// comments, blank lines, and parse errors span the full width. The selected
// row is lifted to the surface tier with an accent bar down its left edge -
// the same selection language the env/file rows use.
func renderEnvRow(entry cmds.EnvEntry, contentWidth int, isSelected, isRevealed bool) string {
	rowBg := chrome.ListRowBg(isSelected)

	var rowContent string
	switch entry.Source {
	case "comment":
		rowContent = lipgloss.NewStyle().
			Foreground(appstyles.Active.TextDim).
			Background(rowBg).
			Width(contentWidth).
			Render(chrome.Truncate(entry.Raw, contentWidth))

	case "parse_error":
		rowContent = lipgloss.NewStyle().
			Foreground(appstyles.Active.Danger).
			Background(rowBg).
			Width(contentWidth).
			Render(chrome.Truncate("[parse error] "+entry.Raw, contentWidth))

	case "var":
		keyWidth, valWidth := keyColWidth(contentWidth)

		keyColor := appstyles.Active.TextPrimary
		if !isSelected {
			keyColor = appstyles.Active.TextMuted
		}

		value := strings.Repeat("•", maskWidth)
		if isRevealed {
			value = entry.Value
		}

		keyCell := lipgloss.NewStyle().
			Foreground(keyColor).
			Background(rowBg).
			Width(keyWidth + 1). // +1 for the gap to the value column
			Render(chrome.Truncate(entry.Key, keyWidth))

		valCell := lipgloss.NewStyle().
			Foreground(appstyles.Active.TextPrimary).
			Background(rowBg).
			Width(valWidth).
			Render(chrome.Truncate(value, valWidth))

		rowContent = lipgloss.JoinHorizontal(lipgloss.Left, keyCell, valCell)

	default: // "blank" and anything unrecognised: a full-width empty row
		rowContent = lipgloss.NewStyle().
			Background(rowBg).
			Width(contentWidth).
			Render("")
	}

	// The accent bar marks the cursor row; other rows reserve the same
	// column in the row background so every row lines up on the same edge.
	barColor := rowBg
	if isSelected {
		barColor = appstyles.Active.Accent
	}
	bar := chrome.BarColumn(barColor, rowBg, rowContent)

	row := lipgloss.JoinHorizontal(lipgloss.Left, bar, rowContent)

	return appstyles.FillBackground(rowBg, row)
}
