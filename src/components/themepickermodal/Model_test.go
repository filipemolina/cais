package themepickermodal

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

// specialKey builds a KeyPressMsg for a special key (esc, enter) where
// Code alone resolves to the right string for key.Matches.
func specialKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

func TestThemePickerStartsOnActiveTheme(t *testing.T) {
	// Reset to a known state.
	appstyles.SetTheme("stitcher-dark")
	defer appstyles.SetTheme("stitcher-dark")

	m := New(40)
	tpm, ok := m.(Model)
	if !ok {
		t.Fatal("New returned unexpected type")
	}

	// The list cursor should be on the active theme.
	item, ok := tpm.list.SelectedItem().(apptypes.ThemeItem)
	if !ok {
		t.Fatal("no item selected")
	}
	if item.Name != "stitcher-dark" {
		t.Errorf("cursor starts on %q, want %q", item.Name, "stitcher-dark")
	}
}

func TestThemePickerEscRestoresOriginalTheme(t *testing.T) {
	appstyles.SetTheme("stitcher-dark")
	defer appstyles.SetTheme("stitcher-dark")

	m := New(40)

	// Move down to preview a different theme.
	for i := 0; i < 2; i++ {
		m, _ = m.Update(specialKey(tea.KeyDown))
	}

	// The active theme has changed from the preview.
	previewTheme := appstyles.Active.Name
	if previewTheme == "stitcher-dark" {
		t.Skip("preview did not move (too few themes?)")
	}

	// Press Esc — should restore the original.
	m, cmd := m.Update(specialKey(tea.KeyEsc))
	if cmd == nil {
		t.Fatal("esc returned no command")
	}

	// The active theme is back to the original.
	if appstyles.Active.Name != "stitcher-dark" {
		t.Errorf("esc left theme at %q, want %q", appstyles.Active.Name, "stitcher-dark")
	}

	// The command should close the modal.
	msg := cmd()
	if _, ok := msg.(cmds.CloseModalMsg); !ok {
		t.Errorf("esc command returned %T, want CloseModalMsg", msg)
	}

	_ = m
}

func TestThemePickerEnterApplies(t *testing.T) {
	appstyles.SetTheme("stitcher-dark")
	defer appstyles.SetTheme("stitcher-dark")

	m := New(40)

	// Move to a different theme.
	m, _ = m.Update(specialKey(tea.KeyDown))

	// The theme under the cursor is now being previewed.
	tpm, _ := m.(Model)
	item, _ := tpm.list.SelectedItem().(apptypes.ThemeItem)
	chosen := item.Name

	// Press Enter.
	_, cmd := m.Update(specialKey(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("enter returned no command")
	}

	// The command chain should include a CloseModal followed by ApplyTheme.
	// Since CloseModal wraps the follow-up, we get a CloseModalMsg with a
	// Follow command that applies the theme.
	msg := cmd()
	closeMsg, ok := msg.(cmds.CloseModalMsg)
	if !ok {
		t.Fatalf("enter command returned %T, want CloseModalMsg", msg)
	}
	if closeMsg.Follow == nil {
		t.Fatal("CloseModal has no follow command")
	}

	// Execute the follow-up: it should apply the chosen theme.
	followMsg := closeMsg.Follow()
	applied, ok := followMsg.(cmds.ThemeAppliedMsg)
	if !ok {
		t.Fatalf("follow command returned %T, want ThemeAppliedMsg", followMsg)
	}
	if applied.Err != nil {
		t.Errorf("ApplyTheme error: %v", applied.Err)
	}
	if applied.Name != chosen {
		t.Errorf("applied theme = %q, want %q", applied.Name, chosen)
	}
}

func TestThemePickerLivePreview(t *testing.T) {
	appstyles.SetTheme("stitcher-dark")
	defer appstyles.SetTheme("stitcher-dark")

	m := New(40)

	// Move down once — the theme should change live.
	m, _ = m.Update(specialKey(tea.KeyDown))

	if appstyles.Active.Name == "stitcher-dark" {
		t.Error("moving the cursor should have previewed a different theme")
	}
}

func TestThemePickerRendersAllThemes(t *testing.T) {
	appstyles.SetTheme("stitcher-dark")
	defer appstyles.SetTheme("stitcher-dark")

	m := New(40)
	tpm, _ := m.(Model)

	if tpm.list.Items() == nil || len(tpm.list.Items()) == 0 {
		t.Fatal("theme picker has no items")
	}

	if len(tpm.list.Items()) != len(appstyles.Themes) {
		t.Errorf("picker has %d items, registry has %d themes",
			len(tpm.list.Items()), len(appstyles.Themes))
	}
}

func TestThemePickerFitsShortTerminal(t *testing.T) {
	appstyles.SetTheme("stitcher-dark")
	defer appstyles.SetTheme("stitcher-dark")

	m := New(20)
	tpm, ok := m.(Model)
	if !ok {
		t.Fatal("New returned unexpected type")
	}

	h := lipgloss.Height(tpm.View().Content)
	if h > 20 {
		t.Errorf("modal rendered height = %d, want <= 20", h)
	}
}

// The Theme binding should appear in the help overlay's Global scope.
func TestThemeKeyInCatalog(t *testing.T) {
	catalog := keys.Catalog(keys.Context{Page: "Home"})

	var found bool
	for _, scope := range catalog {
		if scope.Title != "Global" {
			continue
		}
		for _, entry := range scope.Entries {
			if entry.Binding.Help().Key == "T" {
				found = true
				if !entry.Available {
					t.Error("T theme should be available in the help overlay")
				}
			}
		}
	}
	if !found {
		t.Error("T theme key not found in the help overlay catalog")
	}
}
