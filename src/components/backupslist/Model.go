package backupslist

import (
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/keys"
	"github.com/filipemolina/cais/src/utils"
)

// backupItem is one stored version as a list row.
//
// It lives here rather than in apptypes beside GroupListItem and
// ServiceListItem because it wraps a utils.BackupEntry and utils imports
// apptypes - the reverse dependency would be a cycle. Nothing outside this
// package needs the type, so the wrap costs nothing.
type backupItem struct {
	entry utils.BackupEntry
	// isCurrent marks the copies whose content matches the live file's
	// right now - "this is what you have now". It is decided when the rows
	// are built, from the live hashes the list read carried, so the
	// delegate never has to look past the item.
	isCurrent bool
}

// Title is the live file's own name rather than the "compose" / ".env" tag
// behind it, because compose.yml and compose.yaml are different files and a
// list that calls both "compose" cannot say which one a copy would be
// restored over. See utils.BackupEntry.File.
func (b backupItem) Title() string { return b.entry.File }

// FilterValue is what "/" searches. It carries all three of the things
// someone would filter a version list by - which file, when it was written,
// and which content hash - so "compose", "2026-08-11" and a sha prefix all
// narrow the rows. The other two body lists filter on a name alone because a
// name is all their rows are; a stored version is identified by all three.
func (b backupItem) FilterValue() string {
	return strings.Join([]string{
		b.entry.File,
		b.entry.Timestamp.UTC().Format(timestampLayout),
		b.entry.SHA8,
	}, " ")
}

// Model is the Backups page's left panel: a merged, newest-first list of
// stored versions of the compose file and the .env.
//
// It owns the cursor and nothing else. The bytes behind the selected row
// belong to backuppreviewpanel, which learns about the selection through
// cmds.SetSelectedBackupMsg rather than by being handed a pointer - the two
// panels are siblings under AppModel, the same as Home's list and details.
//
// The rows live in a bubbles list, the same as the groups and services lists,
// which is where the cursor, the pagination and the filter come from. This
// panel hand-rolled all three until it was brought in line; see
// docs/plans/backups-rework.md.
type Model struct {
	list         list.Model
	listDelegate backupsListCustomDelegate
	loading      bool
	loadErr      error
	panelWidth   int
	panelHeight  int
	// focused is whether the arrows are driving this panel rather than the
	// preview's viewport. AppModel owns the answer and broadcasts it; this is
	// the panel's copy, so the panel can ignore a keystroke meant for its
	// sibling without either of them asking the other.
	//
	// Restore is deliberately outside it - see handleKey.
	focused bool
	// rowsRebuilding records that the rows were replaced while a filter was
	// standing, so the cursor is over nothing for one message cycle. See
	// Model.Update's auto-publish.
	rowsRebuilding bool
}

func (m Model) Init() tea.Cmd { return nil }

// OwnsKeyboard reports whether the list is taking every keystroke for itself,
// which it does while a filter is being typed. Same rule as the groups list -
// see groupslist.Model.OwnsKeyboard.
func (m Model) OwnsKeyboard() bool {
	return m.list.FilterState() == list.Filtering
}

// KeepsEsc reports whether the list needs esc for itself. Same rule as the
// groups list - see groupslist.Model.KeepsEsc. It is the only thing esc does
// on this page: there is no selection to clear here, so with no filter
// standing the key is inert and the footer does not offer it.
func (m Model) KeepsEsc() bool {
	return m.list.FilterState() == list.FilterApplied
}

// FilterState exposes how much of the keyboard the list has taken, so
// AppModel can snapshot it into the help overlay's context.
func (m Model) FilterState() list.FilterState {
	return m.list.FilterState()
}

// ListEmpty reports whether there are any stored versions at all, so the
// footer and the help overlay can drop the filter key on an empty store.
// AppModel derives Home's and Services' emptiness from the loaded config; the
// backup store is only known here, so this panel has to answer for itself.
func (m Model) ListEmpty() bool {
	return len(m.list.Items()) == 0
}

// New builds the version list. It starts empty and loading; the page switch
// issues GetBackups, whose result fills it via BackupListMsg.
//
// It starts focused because the page opens on the list: it is the panel with
// the cursor, and the preview has nothing to scroll until that cursor picks a
// row.
func New() tea.Model {
	model := Model{loading: true, focused: true}

	delegate := backupsListCustomDelegate{panelFocused: true}
	backupsList := list.New(nil, delegate, 0, 0)
	backupsList.SetShowHelp(false)

	// Unlike the groups and services lists, the status bar is on whether or
	// not a filter is standing. It is the row that says "5 versions", which
	// this panel showed in its title before the list took the job over, and a
	// version count is worth keeping on screen: it is the answer to "how much
	// history do I have", which the rows alone cannot give once they paginate.
	backupsList.SetShowStatusBar(true)
	backupsList.SetStatusBarItemName("version", "versions")

	// See keys.ListKeyMap: the default map's letter aliases for paging collide
	// with the panel verbs.
	backupsList.KeyMap = keys.ListKeyMap()

	backupsList.Title = "Backups"
	backupsList.Paginator.ActiveDot = " ● "
	backupsList.Paginator.InactiveDot = " ○ "

	model.list = backupsList
	model.listDelegate = delegate

	return model
}

// selected returns the entry under the cursor, or false when the list is
// empty or a filter has left nothing under it.
func (m Model) selected() (utils.BackupEntry, bool) {
	item, ok := m.list.SelectedItem().(backupItem)
	if !ok {
		return utils.BackupEntry{}, false
	}

	return item.entry, true
}

// publishSelection tells the preview panel which copy the cursor is on. It
// is the one place the list speaks to its sibling, so every path that moves
// the cursor goes through it rather than emitting the message by hand.
func (m Model) publishSelection() tea.Cmd {
	entry, ok := m.selected()
	if !ok {
		return cmds.ClearSelectedBackup()
	}

	return cmds.SetSelectedBackup(entry)
}

// resizeList sizes the inner list to the space left inside the panel box
// after the wrapper padding.
func (m *Model) resizeList() {
	h, v := chrome.ListWrapperStyle.GetFrameSize()

	m.list.SetSize(
		max(0, m.panelWidth-h),
		max(0, m.panelHeight-v),
	)
}

// setItems replaces the rows and settles the pagination.
//
// The second pass is not redundant. list.updatePagination subtracts the
// paginator's row from the height it divides into pages, but
// list.paginationView renders nothing - and so measures zero - while
// TotalPages is still 1. Going from no rows to many therefore computes
// PerPage as though there were no paginator, discovers it needs one, and
// leaves the page one row too tall: the last row is drawn under the dots.
// Re-running it once the total is known settles it.
//
// The groups and services lists get their second pass by accident, from the
// SetDelegate inside syncActiveIndex - which only fires when the active row
// actually moved. This list has no activeIndex to sync, so it asks outright.
func (m *Model) setItems(items []list.Item) tea.Cmd {
	cmd := m.list.SetItems(items)
	m.resizeList()

	return cmd
}

// setFocus repoints the delegate at the panel's new tier. The rows sit flush
// on whatever tier the panel is on, and the delegate is the only thing that
// draws them, so the flag has to reach it - see chrome.ListRowBgOn.
func (m *Model) setFocus(focused bool) {
	if m.focused == focused {
		return
	}

	m.focused = focused
	m.listDelegate.panelFocused = focused
	m.list.SetDelegate(m.listDelegate)
}
