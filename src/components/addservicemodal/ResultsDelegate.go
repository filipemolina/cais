package addservicemodal

import (
	"fmt"
	"io"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/utils"
)

type searchResultItem struct{ result utils.ImageResult }

func (i searchResultItem) FilterValue() string { return i.result.Name }

// resultsDelegate renders one search result. This file is a placeholder
// for Step 9 of docs/plans/image-search.md Phase 2A - the full two-line
// render (name + official/stars suffix, dim description below) lands there.
type resultsDelegate struct{ width int }

func (d resultsDelegate) Height() int                             { return 1 }
func (d resultsDelegate) Spacing() int                            { return 0 }
func (d resultsDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d resultsDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(searchResultItem)
	if !ok {
		return
	}

	style := lipgloss.NewStyle().Foreground(appstyles.Active.TextMuted)
	if index == m.Index() {
		style = style.Foreground(appstyles.Active.TextPrimary).Bold(true)
	}
	fmt.Fprint(w, style.Render(item.result.Name))
}
