package themepickermodal

import (
	"fmt"
	"io"
	"slices"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

// themePickerDelegate renders one theme name per row, marking the active
// theme and highlighting the cursor.
type themePickerDelegate struct{}

func (d themePickerDelegate) Height() int                             { return 1 }
func (d themePickerDelegate) Spacing() int                            { return 0 }
func (d themePickerDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d themePickerDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(apptypes.ThemeItem)
	if !ok {
		return
	}

	style := lipgloss.NewStyle().Foreground(appstyles.Active.TextMuted)
	if index == m.Index() {
		style = style.Foreground(appstyles.Active.TextPrimary).Bold(true)
	} else if item.Active {
		style = style.Foreground(appstyles.Active.Accent)
	}

	fmt.Fprint(w, style.Render(item.Title()))
}

// Model is the theme picker: a list of registered themes
// with live preview on cursor movement. Enter applies and persists; Esc
// restores the theme that was active when the modal opened.
type Model struct {
	// originalTheme is the theme name to restore on cancel. Captured once
	// at construction time so Esc always goes back to what the user started
	// with, even after several preview cursor movements.
	originalTheme string
	list          list.Model
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var finalCmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(keyMsg, keys.Overlay.Cancel):
			// Restore the theme the user started with, then close.
			appstyles.SetTheme(m.originalTheme)
			return m, cmds.CloseModal(nil)

		case key.Matches(keyMsg, keys.Overlay.Submit):
			if item, ok := m.list.SelectedItem().(apptypes.ThemeItem); ok {
				// Close first, then apply-and-persist as the follow-up.
				// This matches the compose file picker's pattern:
				// CloseModal(follow) so the modal is gone before the
				// action runs.
				return m, cmds.CloseModal(cmds.ApplyTheme(item.Name))
			}
		}
	}

	// Track the cursor index before the list updates so we can detect
	// movement and preview the new theme live.
	prevIndex := m.list.Index()

	var listCmd tea.Cmd
	m.list, listCmd = m.list.Update(msg)
	finalCmds = append(finalCmds, listCmd)

	// If the cursor moved, preview the theme under it. This repaints the
	// entire UI behind the modal on the next frame - the point of live
	// preview. The original theme is preserved in m.originalTheme so Esc
	// still restores correctly.
	if m.list.Index() != prevIndex {
		if item, ok := m.list.SelectedItem().(apptypes.ThemeItem); ok {
			appstyles.SetTheme(item.Name)
		}
	}

	return m, tea.Batch(finalCmds...)
}

// themePickerHints is the modal's own help line. Two rows: navigation on
// the first, confirm/cancel on the second.
func themePickerHints() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		chrome.RenderKeyHints([]chrome.KeyHint{
			chrome.HintFor(keys.List.Navigate),
		}, appstyles.Active.TextMuted),
		chrome.RenderKeyHints([]chrome.KeyHint{
			chrome.HintAs(keys.Overlay.Submit, "apply"),
			chrome.HintFor(keys.Overlay.Cancel),
		}, appstyles.Active.TextMuted),
	)
}

func (m Model) View() tea.View {
	content := lipgloss.JoinVertical(lipgloss.Left, chrome.ModalTitle("Choose theme"), m.list.View(), "", themePickerHints())

	return tea.NewView(chrome.ModalSurface(appstyles.Active.ModalBg, content))
}

// New builds the theme picker overlay. It lists every
// registered theme, marks the one currently active, and starts the cursor
// on it. Moving the cursor previews that theme live; Enter applies and
// persists; Esc restores the original.
//
// termHeight is the terminal height in rows — used to size the list so it
// never overflows the modal chrome (borders, title, hints).
func New(termHeight int) tea.Model {
	currentTheme := appstyles.Active.Name

	// Collect and sort theme names for a stable order.
	names := make([]string, 0, len(appstyles.Themes))
	for name := range appstyles.Themes {
		names = append(names, name)
	}
	slices.Sort(names)

	items := make([]list.Item, 0, len(names))
	activeIndex := 0
	for i, name := range names {
		items = append(items, apptypes.ThemeItem{Name: name, Active: name == currentTheme})
		if name == currentTheme {
			activeIndex = i
		}
	}

	visible := chrome.ModalListHeight(len(items), termHeight)

	picker := list.New(items, themePickerDelegate{}, 40, visible)
	picker.SetShowTitle(false)
	picker.SetShowHelp(false)
	picker.SetShowStatusBar(false)
	picker.SetShowPagination(visible < len(items))
	picker.SetShowFilter(false)
	picker.Select(activeIndex)

	return Model{
		originalTheme: currentTheme,
		list:          picker,
	}
}
