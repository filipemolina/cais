package model

import (
	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/utils"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Navigating to a page with no components used to return the zero tea.View.
// That leaves AltScreen false, which drops the terminal out of the alternate
// screen buffer: the program keeps running but the UI vanishes, which looks
// exactly like a crash. Every page must render a full frame.
func TestEveryPageRendersAFullFrame(t *testing.T) {
	for _, page := range apptypes.PageTitles {
		t.Run(page, func(t *testing.T) {
			m := applyLayout(drive(startup(120, 40), cmds.SetActivePageMsg(page)))

			view := m.View()

			if !view.AltScreen {
				t.Error("AltScreen is false: the app would leave the alternate screen and appear to exit")
			}

			if strings.TrimSpace(ansi.Strip(view.Content)) == "" {
				t.Fatal("rendered nothing")
			}

			// The frame must fill the terminal, not collapse to the nav and
			// footer alone.
			if got, want := lipgloss.Height(view.Content), 40; got != want {
				t.Errorf("frame height: got %d, want %d", got, want)
			}

			if got, want := lipgloss.Width(view.Content), 120; got != want {
				t.Errorf("frame width: got %d, want %d", got, want)
			}
		})
	}
}

// The pages map is what View, the layout broadcast and the focus cycle all key
// off, so a page listed in the nav but missing from the map is a latent blank
// screen.
func TestEveryNavPageHasComponents(t *testing.T) {
	m := GetInitialModel(utils.ComposeSource{})

	for _, page := range apptypes.PageTitles {
		if len(m.pages[page]) == 0 {
			t.Errorf("page %q is in the nav but has no components registered", page)
		}
	}
}
