package addservicemodal

import (
	"fmt"
	"io"
	"strconv"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/utils"
)

type searchResultItem struct{ result utils.ImageResult }

func (i searchResultItem) FilterValue() string { return i.result.Name }

// resultsDelegate renders one search result over two lines: the name and a
// compact official/stars suffix on the first, the description (dim, always
// truncated to fit) on the second. width is the list's own content width
// (list.Model.Width()), threaded in at construction so every column can be
// sized correctly without recomputing it per row.
type resultsDelegate struct{ width int }

func (d resultsDelegate) Height() int                             { return 2 }
func (d resultsDelegate) Spacing() int                            { return 0 }
func (d resultsDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

// suffixWidth is fixed-width, right-aligned, and holds only what
// distinguishes two same-named results: a star count and, for an official
// image, the plain word "official" - no icon (D9: "●" is this app's only
// precedented symbol, and it already means something else - container
// health/state - so it is not reused here for a different meaning).
const suffixWidth = 16

func (d resultsDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(searchResultItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	nameColor := appstyles.Active.TextMuted
	if isSelected {
		nameColor = appstyles.Active.TextPrimary
	}
	nameStyle := lipgloss.NewStyle().Bold(isSelected).Foreground(nameColor)
	descStyle := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim)

	// Same column technique detailspanel's prop table uses for its
	// PROPERTY/VALUE rows (renderPropRow, src/components/detailspanel/
	// View.go:254-258): a fixed lipgloss.Style.Width per column, each
	// value pre-truncated with chrome.Truncate before it is styled. Do
	// not build this line with fmt.Sprintf field widths ("%-30s") - that
	// pads by byte count, not display width, and silently misaligns as
	// soon as a real Hub description contains a non-ASCII character.
	nameWidth := d.width - suffixWidth
	name := nameStyle.Width(nameWidth).Render(chrome.Truncate(item.result.Name, nameWidth))

	suffix := strconv.Itoa(item.result.Stars) + " stars"
	if item.result.Official {
		suffix = "official · " + suffix
	}
	suffixStyle := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim).Width(suffixWidth).Align(lipgloss.Right)
	suffixCol := suffixStyle.Render(chrome.Truncate(suffix, suffixWidth))

	line1 := lipgloss.JoinHorizontal(lipgloss.Left, name, suffixCol)
	line2 := descStyle.Width(d.width).Render(chrome.Truncate(item.result.Description, d.width))

	fmt.Fprint(w, lipgloss.JoinVertical(lipgloss.Left, line1, line2))
}
