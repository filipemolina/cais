package envmodal

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/components/chrome"
	"github.com/filipemolina/cais/src/keys"
)

// envItem is a parsed .env line as a list row. Comments, blanks and parse
// errors are rows too: the table shows the file, not just its variables, and
// the cursor may land on any of them - selectedVar is what tells them apart.
type envItem struct {
	cmds.EnvEntry
}

// FilterValue is unused: this list does not filter. A .env is short enough to
// read, and / is not among the keys the modal advertises.
func (i envItem) FilterValue() string { return i.Key }

// Model is the env experience as a centered modal: a key/value table of the
// .env file, with reveal, copy, add, edit, delete, an external-editor open,
// and a raw edit, all inside one surface. It replaces the old Env page.
type Model struct {
	envPath string
	// list owns the rows and the cursor. It used to be a plain slice with a
	// selectedIdx beside it, rendered in full every frame - so a .env longer
	// than the modal simply ran off the bottom of it, and the twelve key
	// bindings the cursor needed were built inline on every keystroke.
	list list.Model
	// revealed is the row index whose value is shown in clear text (-1 none).
	// It is keyed by index rather than carried on the item because revealing
	// is a property of this viewing, not of the variable: moving the cursor
	// re-hides it, and reloading the file must not carry it over.
	revealed        int
	loading         bool
	loadErr         error
	parseErrorCount int
	// termHeight is the terminal height in rows. Kept because the list has to
	// be re-fitted when the terminal changes size under an open modal.
	termHeight int
}

func (m Model) Init() tea.Cmd { return nil }

// New builds the env modal for envPath. It does not load on its own: the
// caller (AppModel, on OpenEnvModalMsg) issues GetEnvFileContents so the
// table fills, the same way the page used to.
func New(envPath string, termHeight int) tea.Model {
	return Model{
		envPath:    envPath,
		list:       newList(nil, termHeight),
		revealed:   -1,
		loading:    true,
		termHeight: termHeight,
	}
}

// newList builds the inner list. The keymap is constructed rather than
// defaulted: list.DefaultKeyMap claims q, esc, ?, /, d, f and more, and this
// modal spends most of those itself.
func newList(items []list.Item, termHeight int) list.Model {
	// Two extra rows beyond the usual modal chrome: the KEY/VALUE header and
	// the rule under it.
	visible := chrome.ModalListHeightWith(max(1, len(items)), termHeight, envTableChrome)

	l := list.New(items, envDelegate{revealed: -1}, modalWidth, visible)
	l.KeyMap = keys.ModalListKeyMap()
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(false)
	l.SetShowPagination(visible < len(items))
	l.DisableQuitKeybindings()

	return l
}

// EnvPath returns the .env path the modal is showing.
func (m Model) EnvPath() string { return m.envPath }

// SetEntries fills the table from a parse result.
func (m *Model) SetEntries(path string, entries []cmds.EnvEntry, parseErrors int) {
	items := make([]list.Item, 0, len(entries))
	for _, entry := range entries {
		items = append(items, envItem{entry})
	}

	m.envPath = path
	m.list = newList(items, m.termHeight)
	m.parseErrorCount = parseErrors
	m.loading = false
	m.loadErr = nil
	m.revealed = -1
}

// Entries returns the rows the table is showing, in file order.
func (m Model) Entries() []cmds.EnvEntry {
	out := make([]cmds.EnvEntry, 0, len(m.list.Items()))
	for _, item := range m.list.Items() {
		out = append(out, item.(envItem).EnvEntry)
	}

	return out
}

// SetLoadError records a read failure.
func (m *Model) SetLoadError(err error) {
	m.loadErr = err
	m.loading = false
}

// selectedVar returns the highlighted real variable (source == "var"), or nil
// when the cursor is on a comment/blank/parse-error row or the list is empty.
func (m Model) selectedVar() *cmds.EnvEntry {
	item, ok := m.list.SelectedItem().(envItem)
	if !ok || item.Source != "var" {
		return nil
	}

	entry := item.EnvEntry

	return &entry
}

// envTableChrome is the rows this modal spends above the list beyond what
// chrome.modalListChrome already counts: the KEY / VALUE header and the rule
// under it.
const envTableChrome = 2

// resizeEnvList re-fits the list to a new terminal height, keeping the two
// extra header rows in the arithmetic. chrome.ResizeModalList does not know
// about them.
func resizeEnvList(l *list.Model, termHeight int) {
	items := len(l.Items())
	visible := chrome.ModalListHeightWith(max(1, items), termHeight, envTableChrome)
	l.SetHeight(visible)
	l.SetShowPagination(visible < items)
}
