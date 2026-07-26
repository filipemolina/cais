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
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"

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
	// Page is advertised but not matched: the page chords are recognised by
	// their alt modifier rather than by keystroke, so that alt+shift+g and
	// ctrl+alt+g are left alone. See model.pageForKey.
	Page key.Binding
}

// ListKeys act on the body's left panel: the groups list and the services
// list. New and Delete only mean something on the groups list, which is the
// only list whose contents the app can create.
//
// Filter and ClearFilter belong to the bubbles list rather than to a handler
// here, and are declared anyway so the footer advertises them from the same
// place as everything else. See ListKeyMap.
type ListKeys struct {
	Navigate     key.Binding
	Select       key.Binding
	New          key.Binding
	Delete       key.Binding
	Filter       key.Binding
	ClearFilter  key.Binding
	ApplyFilter  key.Binding
	CancelFilter key.Binding
}

// DetailsKeys act on whatever the body's right panel is showing. The first six
// are shared verbatim between the group panel and the service panel: same key,
// same meaning, one scope wider or narrower. EditService and EditFile exist
// only on the service panel, which is the only place a single service is the
// subject.
type DetailsKeys struct {
	Start       key.Binding
	Stop        key.Binding
	Restart     key.Binding
	Pull        key.Binding
	Remove      key.Binding
	Logs        key.Binding
	EditService key.Binding
	EditFile    key.Binding
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
	// Not advertised: ctrl+c is the escape hatch every terminal program has,
	// and the footer already says q. It carries no help text so Globals stays
	// the same two hints it was.
	ForceQuit: key.NewBinding(key.WithKeys("ctrl+c")),
	Page:      key.NewBinding(key.WithHelp("alt+·", "page")),
}

var List = ListKeys{
	// Matched by the bubbles list itself; declared here so the footer can
	// advertise it from the same place as everything else.
	Navigate:    key.NewBinding(key.WithHelp("↑/↓", "navigate")),
	Select:      key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "select")),
	New:         key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
	Delete:      key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
	Filter:      key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	ClearFilter: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear filter")),
	// The same keystrokes as Overlay.Submit and Overlay.Cancel, because a
	// filtering list is an overlay. They are declared separately only so the
	// help text can say what enter and esc do to a filter rather than the
	// generic confirm/cancel a modal shows.
	ApplyFilter:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "apply")),
	CancelFilter: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
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
		GoToStart: key.NewBinding(key.WithKeys("home", "g"), key.WithHelp("g/home", "go to start")),
		GoToEnd:   key.NewBinding(key.WithKeys("end", "G"), key.WithHelp("G/end", "go to end")),

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
			// made - but Delete needs something to delete.
			own := []key.Binding{List.New}
			if !ctx.ListEmpty {
				own = append(own, List.Delete)
			}

			return listKeys(ctx, own...)

		case constants.COMPONENT_BODY_DETAILS:
			if !ctx.Selected {
				return []key.Binding{Global.NextPanel}
			}

			return []key.Binding{
				Details.Start, Details.Stop, Details.Restart,
				Details.Pull, Details.Remove, Details.Logs,
				Global.NextPanel,
			}
		}

	case "Services":
		switch ctx.Focused {
		case constants.COMPONENT_BODY_LIST:
			// The services list is read-only: services are created by editing
			// the compose file, not from here.
			return listKeys(ctx)

		case constants.COMPONENT_BODY_DETAILS:
			if !ctx.Selected {
				return []key.Binding{Global.NextPanel}
			}

			return []key.Binding{
				Details.Start, Details.Stop, Details.Restart,
				Details.Pull, Details.Remove, Details.Logs,
				Details.EditService, Details.EditFile,
				Global.NextPanel,
			}
		}

	// Pages with nothing focusable would otherwise advertise a Tab that
	// visibly does nothing.
	case "Compose Files":
		return nil
	}

	return []key.Binding{Global.NextPanel}
}

// Globals are the always-available keys the footer pins to its right-hand side,
// away from the context-dependent ones.
func Globals() []key.Binding {
	return []key.Binding{Global.Page, Global.Quit}
}
