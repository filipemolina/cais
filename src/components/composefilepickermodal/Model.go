package composefilepickermodal

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
)

type Model struct {
	dir  string
	list list.Model
}

func (m Model) Init() tea.Cmd {
	return nil
}

// New lists the YAML files in dir for switching the active compose file.
// activeName (the base name of the loaded file) is marked, and the cursor
// starts on it. Enter switches to the highlighted file, Esc cancels.
//
// termHeight is the terminal height in rows - a directory can hold more
// compose files than the screen has room for, so the list is sized to fit
// rather than to len(items). See chrome.ModalListHeight.
func New(dir string, fileNames []string, activeName string, termHeight int) tea.Model {
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

	return Model{
		dir:  dir,
		list: picker,
	}
}
