package model

import (
	"stack-stitcher/src/cmds"
	"stack-stitcher/src/constants"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// drive feeds messages through the model the way the Bubble Tea loop would,
// discarding the commands. Layout is driven entirely by the messages listed
// here, so a test can reproduce any startup or resize ordering.
func drive(m tea.Model, msgs ...tea.Msg) AppModel {
	for _, msg := range msgs {
		m, _ = m.Update(msg)
	}

	return m.(AppModel)
}

// layoutMsg is the layout AppModel would broadcast in its current state.
func layoutMsg(m AppModel) cmds.SetBodyLayoutMsg {
	return m.config.bodyLayout
}

// startup replays the relevant startup sequence. The window size arrives
// before SetActivePageMsg, which is the ordering that used to leave every
// panel at width 0: WindowSizeMsg only reaches the active page's components,
// and no page is active yet. Activating Home then emits the focus and layout
// messages that the runtime routes back into the model.
func startup(width, height int) AppModel {
	m := drive(GetInitialModel(), tea.WindowSizeMsg{Width: width, Height: height})
	updated, cmd := m.Update(cmds.SetActivePageMsg("Home"))

	return drive(updated, collect(cmd)...)
}

// applyLayout hands the model's own layout back to its components, standing
// in for the command round-trip the runtime performs.
func applyLayout(m AppModel) AppModel {
	return drive(m, layoutMsg(m))
}

func TestBodyLayoutFillsTerminalWidthExactly(t *testing.T) {
	sizes := []struct{ width, height int }{
		{120, 40},
		{80, 24},
		{200, 60},
		{64, 20},  // narrower than two minimum panels: split evenly
		{300, 80}, // wide
	}

	for _, size := range sizes {
		layout := layoutMsg(startup(size.width, size.height))
		total := layout.LeftWidth + constants.BODY_GUTTER_WIDTH + layout.RightWidth

		if total != size.width {
			t.Errorf("terminal %dx%d: left(%d) + gutter(%d) + right(%d) = %d, want %d",
				size.width, size.height,
				layout.LeftWidth, constants.BODY_GUTTER_WIDTH, layout.RightWidth,
				total, size.width)
		}

		if layout.Height <= 0 {
			t.Errorf("terminal %dx%d: body height %d, want > 0", size.width, size.height, layout.Height)
		}
	}
}

func TestPanelsRenderAtTheirBroadcastSize(t *testing.T) {
	for _, page := range []string{"Home", "Dashboard"} {
		m := applyLayout(drive(startup(120, 40), cmds.SetActivePageMsg(page)))
		layout := layoutMsg(m)

		want := []int{layout.LeftWidth, layout.RightWidth}
		for idx, component := range m.pages[page] {
			content := component.View().Content

			if got := lipgloss.Width(content); got != want[idx] {
				t.Errorf("%s panel %d: width %d, want %d", page, idx, got, want[idx])
			}

			if got := lipgloss.Height(content); got != layout.Height {
				t.Errorf("%s panel %d: height %d, want %d", page, idx, got, layout.Height)
			}
		}
	}
}

// The Dashboard is never the active page when the terminal is first measured,
// so its components only ever learn their size from the layout broadcast that
// follows the page switch.
func TestPanelsAreSizedAfterSwitchingPageWithoutAResize(t *testing.T) {
	m := applyLayout(drive(startup(120, 40), cmds.SetActivePageMsg("Dashboard")))

	for idx, component := range m.pages["Dashboard"] {
		if got := lipgloss.Width(component.View().Content); got < constants.MIN_PANEL_WIDTH {
			t.Errorf("dashboard panel %d: width %d, want >= %d (never received its size)",
				idx, got, constants.MIN_PANEL_WIDTH)
		}
	}
}

func TestRenderedViewNeverExceedsTerminalWidth(t *testing.T) {
	sizes := []struct{ width, height int }{{120, 40}, {80, 24}, {64, 20}}

	for _, size := range sizes {
		for _, page := range []string{"Home", "Dashboard"} {
			m := applyLayout(drive(startup(size.width, size.height), cmds.SetActivePageMsg(page)))

			for i, line := range strings.Split(m.View().Content, "\n") {
				if got := lipgloss.Width(line); got > size.width {
					t.Errorf("%s at %dx%d: line %d is %d wide, want <= %d",
						page, size.width, size.height, i, got, size.width)
					break
				}
			}
		}
	}
}
