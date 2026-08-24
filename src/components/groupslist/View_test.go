package groupslist

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/cmds"
)

// The list sizes a page by the delegate's Height, then prints whatever Render
// returns. When the two disagree the list pages early and pads the shortfall
// out below the last row, which is the gap this guards against: the row lost
// its description line and Height stayed at 4, costing two items a page and
// leaving eight dead rows above the paginator.
func TestDelegateHeightMatchesTheRow(t *testing.T) {
	delegate := GroupsListCustomDelegate{activeIndex: -1}
	inner := list.New([]list.Item{apptypes.GroupListItem{Name: "group"}}, delegate, 40, 20)

	var row strings.Builder
	delegate.Render(&row, inner, 0, apptypes.GroupListItem{Name: "group"})

	if got := lipgloss.Height(row.String()); got != delegate.Height() {
		t.Errorf("Render emits %d rows but Height reports %d", got, delegate.Height())
	}
}

// The panel must spend the height it was given on groups rather than on empty
// rows above the paginator. Rows are a fixed three lines tall, so a panel
// whose height is not a multiple of that leaves a remainder no group could
// have used - what this rules out is dead space large enough to have held
// another group, which is what an over-reported delegate Height produces.
func TestTheListFillsThePanelHeight(t *testing.T) {
	const paginatorMarginTop = 1

	rowHeight := GroupsListCustomDelegate{}.Height()

	for _, panelHeight := range []int{20, 22, 24, 26, 28, 30} {
		var model tea.Model = New(nil, 40, panelHeight)
		for _, msg := range []tea.Msg{
			cmds.SetBodyLayoutMsg{LeftWidth: 40, Height: panelHeight},
			groupNames(12),
			cmds.SetHomeStatsMsg{Groups: 12, Services: 17, Running: 6},
		} {
			model, _ = model.Update(msg)
		}

		groups, ok := model.(Model)
		if !ok {
			t.Fatalf("expected a Model, got %T", model)
		}

		lines := strings.Split(groups.View().Content, "\n")

		lastRow, dots := -1, -1
		for i, line := range lines {
			switch {
			case strings.Contains(line, "group-"):
				lastRow = i
			case strings.Contains(line, "●") && dots < 0 && lastRow >= 0:
				dots = i
			}
		}
		if lastRow < 0 || dots < 0 {
			t.Fatalf("panel height %d: found no group rows (%d) or paginator (%d)", panelHeight, lastRow, dots)
		}

		// The last row carries its own bottom padding line, so the first row
		// that could hold a group is two below the last title line.
		gap := dots - (lastRow + 2)
		if want := paginatorMarginTop + rowHeight - 1; gap > want {
			t.Errorf("panel height %d: %d blank rows between the last group and the paginator, want at most %d - the slack could have held another group",
				panelHeight, gap, want)
		}
	}
}
