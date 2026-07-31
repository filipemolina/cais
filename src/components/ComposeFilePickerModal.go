package components

import (
	"fmt"
	"io"
	"path/filepath"

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

type composeFilePickerDelegate struct{}

func (d composeFilePickerDelegate) Height() int                             { return 1 }
func (d composeFilePickerDelegate) Spacing() int                            { return 0 }
func (d composeFilePickerDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d composeFilePickerDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(apptypes.ComposeFileItem)
	if !ok {
		return
	}

	style := lipgloss.NewStyle().Foreground(appstyles.Active.TextMuted)
	if index == m.Index() {
		style = style.Foreground(appstyles.Active.TextPrimary).Bold(true)
	} else if item.Active {
		// The file already loaded is worth marking even off the cursor, so
		// "switch to the one I'm on" reads as a no-op rather than a choice.
		style = style.Foreground(appstyles.Active.Accent)
	}

	fmt.Fprint(w, style.Render(item.Title()))
}

type ComposeFilePickerModalModel struct {
	dir  string
	list list.Model
}

func (m ComposeFilePickerModalModel) Init() tea.Cmd {
	return nil
}

func (m ComposeFilePickerModalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var finalCmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(keyMsg, keys.Overlay.Cancel):
			return m, cmds.CloseModal(nil)

		case key.Matches(keyMsg, keys.Overlay.Submit):
			if item, ok := m.list.SelectedItem().(apptypes.ComposeFileItem); ok {
				return m, cmds.CloseModal(cmds.SwitchComposeFile(filepath.Join(m.dir, item.Name)))
			}
		}
	}

	var listCmd tea.Cmd
	m.list, listCmd = m.list.Update(msg)
	finalCmds = append(finalCmds, listCmd)

	return m, tea.Batch(finalCmds...)
}

// pickerHints is the modal's own help line; the footer is hidden behind the
// modal, so the keys it takes over are advertised here. Two lines, like the
// checklist modal's, so the modal stays as narrow as its list.
func pickerHints() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		chrome.RenderKeyHints([]chrome.KeyHint{
			chrome.HintFor(keys.List.Navigate),
		}, appstyles.Active.TextMuted),
		chrome.RenderKeyHints([]chrome.KeyHint{
			chrome.HintAs(keys.Overlay.Submit, "switch file"),
			chrome.HintFor(keys.Overlay.Cancel),
		}, appstyles.Active.TextMuted),
	)
}

func (m ComposeFilePickerModalModel) View() tea.View {
	content := lipgloss.JoinVertical(lipgloss.Left, chrome.ModalTitle("Switch compose file"), m.list.View(), "", pickerHints())

	return tea.NewView(chrome.ModalSurface(appstyles.Active.ModalBg, content))
}

// ComposeFilePickerModal lists the YAML files in dir for switching the
// active compose file. activeName (the base name of the loaded file) is
// marked, and the cursor starts on it. Enter switches to the highlighted
// file, Esc cancels.
//
// termHeight is the terminal height in rows - a directory can hold more
// compose files than the screen has room for, so the list is sized to fit
// rather than to len(items). See chrome.ModalListHeight.
func ComposeFilePickerModal(dir string, fileNames []string, activeName string, termHeight int) tea.Model {
	items := make([]list.Item, 0, len(fileNames))
	activeIndex := 0
	for i, name := range fileNames {
		items = append(items, apptypes.ComposeFileItem{Name: name, Active: name == activeName})
		if name == activeName {
			activeIndex = i
		}
	}

	// The title is rendered by chrome.ModalTitle in the View function, not by the
	// list itself. Pagination is off while every file fits - the paginator
	// would cost a row the list has no use for - and on when it doesn't, so
	// the files past the fold stay reachable and are visibly there.
	visible := chrome.ModalListHeight(len(items), termHeight)

	picker := list.New(items, composeFilePickerDelegate{}, 40, visible)
	picker.SetShowTitle(false)
	picker.SetShowHelp(false)
	picker.SetShowStatusBar(false)
	picker.SetShowPagination(visible < len(items))
	picker.SetShowFilter(false)
	picker.Select(activeIndex)

	return ComposeFilePickerModalModel{
		dir:  dir,
		list: picker,
	}
}
