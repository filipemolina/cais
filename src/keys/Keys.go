// Package keys is the single source of truth for every keybinding in the app.
//
// Before this package, a key lived in two places: the component that handled
// it, and the hand-written list in the footer bar that advertised it. Nothing
// held the two together, so the bar could promise a key that no handler
// implemented, or stay silent about one that did.
//
// Two rules follow from that, and they are the reason this package exists:
//
//  1. A key is declared here exactly once. Components match against these
//     bindings; the footer and the help overlay render from them.
//  2. One verb is one binding. Start is the same key on the group panel and
//     the service panel because both read Details.Start - not because two
//     switch statements happen to agree.
//
// The bindings carry their own help text, which is what the footer prints.
package keys

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"

	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/constants"
)

// GlobalKeys work anywhere that no overlay owns the keyboard.
type GlobalKeys struct {
	NextPanel key.Binding
	PrevPanel key.Binding
	// Quit is q, and it is the one global key that yields: a modal or a
	// filtering list needs the letter for typing. ForceQuit is separate from it
	// precisely so that ctrl+c yields to nothing.
	Quit      key.Binding
	ForceQuit key.Binding
	// Back is esc away from everything that has a stronger claim on it: a
	// modal, a filter being typed, a filter standing on a focused list. What
	// is left is the details panel, so esc there means "back to the list" -
	// which is why the footer offers it in the details contexts and nowhere
	// else. See model.AppModel.escKept.
	Back key.Binding
	// Help opens the help overlay. The overlay renders from this package, so
	// what it says is what the handlers do.
	Help key.Binding
	// About opens the About modal: the brand mark, version, license and repo
	// link. A read-only overlay like Help, closed by the same three keys.
	About key.Binding
	// Theme opens the theme picker: a list of registered themes with live
	// preview on cursor movement and persist-on-confirm. T (shift+t) so it
	// does not collide with the details panel's lowercase t (stop).
	Theme key.Binding
	// Page is advertised but not matched: the digits are recognised by their
	// key code and the alt+<letter> alias by its modifier, so that 1 as filter
	// text and alt+shift+g are both left alone. See model.pageForNavKey. The
	// bracket pair steps through the pages in order; it is not in the footer's
	// global group for width, but lives here so the help overlay renders it
	// from the same place as everything else.
	Page     key.Binding
	NextPage key.Binding
	PrevPage key.Binding
}

// ListKeys act on the body's left panel: the groups list and the services
// list. New, Edit and Delete only mean something on the groups list, which
// is the only list whose contents the app can modify. The services list is
// read-only; its services are created by editing the compose file.
//
// Filter, ClearFilter, GoToStart and GoToEnd belong to the bubbles list
// rather than to a handler here, and are declared anyway so the footer and
// the help overlay advertise them from the same place as everything else.
// See ListKeyMap.
type ListKeys struct {
	Navigate     key.Binding
	Select       key.Binding
	New          key.Binding
	Edit         key.Binding
	Delete       key.Binding
	Filter       key.Binding
	ClearFilter  key.Binding
	ApplyFilter  key.Binding
	CancelFilter key.Binding
	GoToStart    key.Binding
	GoToEnd      key.Binding
}

// DetailsKeys act on whatever the body's right panel is showing. The first six
// are shared verbatim between the group panel and the service panel: same key,
// same meaning, one scope wider or narrower. EditService and EditFile exist
// only on the service panel, which is the only place a single service is the
// subject. Save and OpenEditor are only live while the inline editor is open.
type DetailsKeys struct {
	Start       key.Binding
	Stop        key.Binding
	Restart     key.Binding
	Pull        key.Binding
	Remove      key.Binding
	Logs        key.Binding
	EditService key.Binding
	EditFile    key.Binding
	Save        key.Binding
	OpenEditor  key.Binding
}

// EditorKeys act inside the inline YAML editor, and only there. The editor
// owns the whole keyboard while it is open (see DetailsPanelModel.OwnsKeyboard),
// which is what makes tab and shift+tab available here at all - they are the
// panel-switching keys everywhere else, and the app stands down from them
// while the editor holds the keyboard.
type EditorKeys struct {
	NewLine key.Binding
	Indent  key.Binding
	Outdent key.Binding
}

// FilesKeys act on the Files page's read-only file viewer. Scroll is the
// viewport's own (the viewport answers the keystrokes); it is declared here
// so the footer and the help overlay advertise it from the same place as
// everything else - the same pattern as the list's navigation keys. The
// viewer's edit key is Details.EditFile, reused rather than redeclared.
type FilesKeys struct {
	Scroll key.Binding
	Browse key.Binding
}

// OverlayKeys are the keys every modal answers to. Cancel is one binding for
// every overlay in the app, including the logs viewer, so "esc backs out" needs
// no exceptions.
type OverlayKeys struct {
	Submit    key.Binding
	Cancel    key.Binding
	NextField key.Binding
	Toggle    key.Binding
	Yes       key.Binding
	No        key.Binding
	Follow    key.Binding
}

var Global = GlobalKeys{
	NextPanel: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next")),
	PrevPanel: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev")),
	Quit:      key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	// Not in the footer's global group: ctrl+c is the escape hatch every
	// terminal program has, and the footer already says q. The help overlay
	// is where it is advertised.
	ForceQuit: key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "force quit")),
	// The digit range is derived from the page list rather than written out,
	// so a fourth tab extends the hint instead of drifting from it.
	Page: key.NewBinding(
		key.WithHelp(fmt.Sprintf("1-%d", len(apptypes.PageTitles)), "page"),
	),
	NextPage: key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "next page")),
	PrevPage: key.NewBinding(key.WithKeys("["), key.WithHelp("[", "prev page")),
	Back:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	About:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "about")),
	Theme:    key.NewBinding(key.WithKeys("T"), key.WithHelp("T", "theme")),
}

var List = ListKeys{
	// Matched by the bubbles list itself; declared here so the footer can
	// advertise it from the same place as everything else.
	Navigate: key.NewBinding(key.WithHelp("↑/↓", "navigate")),
	// Enter is an alias for space: same verb, same binding, so every panel
	// matches either. The help advertises space alone - the alias is for the
	// muscle memory that expects enter to choose, not another key to learn.
	Select:      key.NewBinding(key.WithKeys("space", "enter"), key.WithHelp("space", "start")),
	New:         key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
	Edit:        key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
	Delete:      key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
	Filter:      key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	ClearFilter: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear filter")),
	// The same keystrokes as Overlay.Submit and Overlay.Cancel, because a
	// filtering list is an overlay. They are declared separately only so the
	// help text can say what enter and esc do to a filter rather than the
	// generic confirm/cancel a modal shows.
	ApplyFilter:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "apply")),
	CancelFilter: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	GoToStart:    key.NewBinding(key.WithKeys("home", "g"), key.WithHelp("g", "first row")),
	GoToEnd:      key.NewBinding(key.WithKeys("end", "G"), key.WithHelp("G", "last row")),
}

var Details = DetailsKeys{
	Start:       key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "start")),
	Stop:        key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "stop")),
	Restart:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "restart")),
	Pull:        key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pull")),
	Remove:      key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "remove")),
	Logs:        key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "logs")),
	EditService: key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
	EditFile:    key.NewBinding(key.WithKeys("E"), key.WithHelp("E", "file")),
	Save:        key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save")),
	OpenEditor:  key.NewBinding(key.WithKeys("ctrl+o"), key.WithHelp("ctrl+o", "editor")),
}

var Editor = EditorKeys{
	// Matched by the editor's own handler, which indents the new line rather
	// than letting the textarea insert a bare newline.
	NewLine: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "new line")),
	Indent:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "indent")),
	Outdent: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "outdent")),
}

var Files = FilesKeys{
	// Help-only, like List.Navigate: the viewport owns the keystrokes, this
	// declares what the footer and the help overlay say about them.
	Scroll: key.NewBinding(key.WithHelp("↑/↓", "scroll")),
	Browse: key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "browse")),
}

var Overlay = OverlayKeys{
	Submit:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
	Cancel:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	NextField: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field")),
	Toggle:    key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "toggle")),
	Yes:       key.NewBinding(key.WithKeys("y", "Y"), key.WithHelp("y", "yes")),
	No:        key.NewBinding(key.WithKeys("n", "N"), key.WithHelp("n", "no")),
	Follow:    key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "follow")),
}

// ListKeyMap is the keymap the two body lists install on their inner bubbles
// list, replacing list.DefaultKeyMap.
//
// The default map is written for a list that is the whole program, so it claims
// keys this app spends elsewhere: d and f page forward while d deletes a group,
// h, l, b and u page while l opens logs, and q, esc and ? are the app's. The
// results were visible - pressing d both opened the delete confirm and paged
// the list backwards - so the list has to be told which keys are not its own.
//
// What stays is what only the list can answer: where its cursor is, and the
// filter. Filtering is worth keeping with forty services, and while it is
// active the list owns the keyboard the way a modal does.
//
// Keys the app owns are left with no keystrokes rather than removed, because
// list.Model reads every field: an empty binding matches nothing, which is the
// intent, whereas a missing one would be a nil-safe accident.
func ListKeyMap() list.KeyMap {
	unbound := key.NewBinding()

	return list.KeyMap{
		CursorUp:   key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		CursorDown: key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		// pgup/pgdown and the arrows only; the letter aliases (h/l/b/u/f/d)
		// are the ones that collided with the panel verbs.
		PrevPage:  key.NewBinding(key.WithKeys("left", "pgup"), key.WithHelp("←/pgup", "prev page")),
		NextPage:  key.NewBinding(key.WithKeys("right", "pgdown"), key.WithHelp("→/pgdn", "next page")),
		GoToStart: List.GoToStart,
		GoToEnd:   List.GoToEnd,

		Filter:      List.Filter,
		ClearFilter: List.ClearFilter,

		// The filtering list is an overlay, so it answers the overlay keystrokes:
		// enter confirms and esc cancels, exactly as they do in every modal. The
		// default map also accepted on tab and the arrows, which made tab both
		// apply the filter and move panels.
		CancelWhileFiltering: List.CancelFilter,
		AcceptWhileFiltering: List.ApplyFilter,

		// The app's, all of them: ? opens the help overlay, q and ctrl+c quit
		// through AppModel, and esc above is the only esc the list needs.
		ShowFullHelp:  unbound,
		CloseFullHelp: unbound,
		Quit:          unbound,
		ForceQuit:     unbound,
	}
}

// Context is what the footer knows about the screen: enough to decide which
// bindings are live, and nothing more.
type Context struct {
	Page      string
	Focused   int
	ListEmpty bool
	// Selected reports whether the panel has a subject to act on - a chosen
	// group on Home, a chosen service on Services. Without one, the action
	// keys do nothing and are not offered.
	Selected bool
	// Editing is true when the service details panel is in inline edit mode.
	// The editor owns the keyboard, so the panel's action keys and the page
	// digits are dead; the footer shows the editor-specific keys instead.
	Editing bool
	// PendingAction is true when a docker action is running. Action keys are
	// disabled and a spinner is shown in the panel.
	PendingAction bool
	// Filter is the focused list's filter state. Its zero value is
	// list.Unfiltered, so a caller that has no list to report about gets the
	// ordinary keys.
	Filter list.FilterState
}

// listKeys are the left panel's keys in the order the footer shows them, with
// the filter slot resolved: while a filter is being typed the list has the
// keyboard and nothing else is pressable, and once one is applied the slot
// becomes the esc that clears it - a key that would otherwise go unmentioned.
func listKeys(ctx Context, own ...key.Binding) []key.Binding {
	if ctx.Filter == list.Filtering {
		return []key.Binding{List.ApplyFilter, List.CancelFilter}
	}

	bindings := make([]key.Binding, 0, len(own)+4)
	if !ctx.ListEmpty {
		bindings = append(bindings, List.Select)
	}
	bindings = append(bindings, own...)

	switch {
	case ctx.Filter == list.FilterApplied:
		bindings = append(bindings, List.ClearFilter)
	case !ctx.ListEmpty:
		// Nothing to filter in an empty list.
		bindings = append(bindings, List.Filter)
	}

	return append(bindings, List.Navigate, Global.NextPanel)
}

// Active returns the bindings the user can press right now, in the order they
// should be shown.
//
// It returns a filtered slice rather than disabling bindings in place:
// key.Binding.Enabled gates matching as well as help, and these bindings are
// package-level values shared with the components, so disabling one to tidy the
// footer would stop the key working everywhere.
func Active(ctx Context) []key.Binding {
	switch ctx.Page {
	case "Home":
		switch ctx.Focused {
		case constants.COMPONENT_BODY_LIST:
			// New is offered even with no groups - it is how the first one gets
			// made - but Edit and Delete need something to act on.
			own := []key.Binding{List.New}
			if !ctx.ListEmpty {
				own = append(own, List.Edit, List.Delete)
			}

			return listKeys(ctx, own...)

		case constants.COMPONENT_BODY_DETAILS:
			if !ctx.Selected {
				return []key.Binding{List.New, Global.Back, Global.NextPanel}
			}

			// While an action is pending, disable action keys.
			if ctx.PendingAction {
				return []key.Binding{Global.Back, Global.NextPanel}
			}

			return []key.Binding{
				List.New,
				Details.Start, Details.Stop, Details.Restart,
				Details.Pull, Details.Remove, Details.Logs,
				Global.Back, Global.NextPanel,
			}
		}

	case "Services":
		switch ctx.Focused {
		case constants.COMPONENT_BODY_LIST:
			// The services list is read-only: services are created by editing
			// the compose file, not from here.
			return listKeys(ctx)

		case constants.COMPONENT_BODY_DETAILS:
			if ctx.Editing {
				return []key.Binding{
					Details.Save, Details.OpenEditor,
					Editor.Indent, Editor.Outdent,
					Global.Back,
				}
			}

			if !ctx.Selected {
				return []key.Binding{Global.Back, Global.NextPanel}
			}

			return []key.Binding{
				Details.Start, Details.Stop, Details.Restart,
				Details.Pull, Details.Remove, Details.Logs,
				Details.EditService, Details.EditFile,
				Global.Back, Global.NextPanel,
			}
		}

	// The Files page has one always-focused panel, so the same keys apply
	// regardless of which component id Tab last touched.
	case "Compose Files":
		return []key.Binding{Details.EditFile, Files.Browse, Files.Scroll}
	}

	return []key.Binding{Global.NextPanel}
}

// Live reports whether a binding is pressable in ctx. It is Active's answer for
// a single binding, for callers that render one control per key rather than a
// list of them - the details panels' action buttons. Going through Active is
// the point: a button is dim exactly when the footer has stopped offering the
// key, because both read the same decision.
func Live(ctx Context, binding key.Binding) bool {
	return containsBinding(Active(ctx), binding)
}

// Globals are the always-available keys the footer pins to its right-hand side,
// away from the context-dependent ones.
func Globals() []key.Binding {
	return []key.Binding{Global.Page, Global.Help, Global.Quit}
}

// Scope is one group of related keys in the help overlay.
type Scope struct {
	Title   string
	Entries []Entry
}

// Entry is one row of the help overlay: the binding to render, and whether
// the user can press it right now. Rows that cannot be pressed are dimmed.
type Entry struct {
	Binding   key.Binding
	Available bool
}

// Catalog returns every key in the app, grouped by scope, with availability
// resolved against ctx. It reads the same bindings the handlers match against
// - that is the point of the overlay: it cannot drift from the handlers.
func Catalog(ctx Context) []Scope {
	live := pressableNow(ctx)

	entries := func(bindings ...key.Binding) []Entry {
		out := make([]Entry, 0, len(bindings))
		for _, b := range bindings {
			out = append(out, Entry{Binding: b, Available: containsBinding(live, b)})
		}
		return out
	}

	// g/G live wherever the arrows do - a focused list - but the footer never
	// advertises them, so Active never returns them.
	listNavigable := containsBinding(live, List.Navigate)

	return []Scope{
		{
			Title: "Pages",
			Entries: append(
				entries(Global.Page, Global.PrevPage, Global.NextPage),
				// The alt chords are always live as aliases; one entry lists
				// them, derived from the labels so it cannot drift either.
				Entry{Binding: pageChordBinding(), Available: true},
			),
		},
		{
			Title: "List",
			Entries: append(
				entries(List.Select, List.New, List.Edit, List.Delete, List.Filter, List.ClearFilter, List.Navigate),
				Entry{Binding: List.GoToStart, Available: listNavigable},
				Entry{Binding: List.GoToEnd, Available: listNavigable},
			),
		},
		{
			Title: "Details",
			Entries: entries(
				Details.Start, Details.Stop, Details.Restart,
				Details.Pull, Details.Remove, Details.Logs,
				Details.EditService, Details.EditFile,
				Details.Save, Details.OpenEditor,
			),
		},
		{
			// Only reachable with the inline editor open, so every row here is
			// dimmed everywhere else - which is the overlay saying "these are the
			// editor's keys" without a sentence of prose.
			Title: "Editor",
			Entries: entries(
				Editor.NewLine, Editor.Indent, Editor.Outdent,
				Details.Save, Details.OpenEditor,
			),
		},
		{
			Title: "Files",
			Entries: entries(
				Details.EditFile, Files.Browse, Files.Scroll,
			),
		},
		{
			// Overlay keys do nothing on the main screen the overlay was
			// opened from, so they are dimmed there by construction.
			Title: "Overlays",
			Entries: entries(
				Overlay.Submit, Overlay.Cancel, Overlay.Yes, Overlay.No,
				Overlay.NextField, Overlay.Toggle, Overlay.Follow,
			),
		},
		{
			Title: "Global",
			Entries: entries(
				Global.NextPanel, Global.PrevPanel, Global.Back,
				Global.Quit, Global.ForceQuit, Global.Help, Global.About,
				Global.Theme,
			),
		},
	}
}

// pressableNow is the set of bindings the user can actually press in ctx: the
// contextual ones Active returns, plus the globals that are always live
// whether or not the footer has room to advertise them.
func pressableNow(ctx Context) []key.Binding {
	live := append(Active(ctx), Globals()...)
	live = append(live, Global.ForceQuit, Global.PrevPage, Global.NextPage, Global.About, Global.Theme)

	// shift+tab is tab's twin: live wherever tab is, with no footer slot of
	// its own.
	if containsBinding(live, Global.NextPanel) {
		live = append(live, Global.PrevPanel)
	}

	return live
}

// pageChordBinding is the help face of the alt+letter aliases: one entry
// listing each page's chord ("alt+g/s/f"), derived from the labels so a
// renamed tab cannot leave it pointing at the old letter.
func pageChordBinding() key.Binding {
	letters := make([]string, 0, len(apptypes.PageTitles))
	for _, page := range apptypes.PageTitles {
		letters = append(letters, apptypes.PageShortcut(page))
	}

	return key.NewBinding(key.WithHelp("alt+"+strings.Join(letters, "/"), "page (alias)"))
}

// sameBinding reports whether two bindings are the same one: same keystrokes,
// same help. The catalog compares values rather than pointers because
// bindings travel by value.
func sameBinding(a, b key.Binding) bool {
	return slices.Equal(a.Keys(), b.Keys()) && a.Help() == b.Help()
}

func containsBinding(haystack []key.Binding, needle key.Binding) bool {
	return slices.ContainsFunc(haystack, func(b key.Binding) bool { return sameBinding(b, needle) })
}
