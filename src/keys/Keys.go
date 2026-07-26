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

	"github.com/filipemolina/stack-stitcher/src/constants"
)

// GlobalKeys work anywhere that no overlay owns the keyboard.
type GlobalKeys struct {
	NextPanel key.Binding
	PrevPanel key.Binding
	Quit      key.Binding
	// Page is advertised but not matched: the page chords are recognised by
	// their alt modifier rather than by keystroke, so that alt+shift+g and
	// ctrl+alt+g are left alone. See model.pageForKey.
	Page key.Binding
}

// ListKeys act on the body's left panel: the groups list and the services
// list. New and Delete only mean something on the groups list, which is the
// only list whose contents the app can create.
type ListKeys struct {
	Navigate key.Binding
	Select   key.Binding
	New      key.Binding
	Delete   key.Binding
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
	Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Page:      key.NewBinding(key.WithHelp("alt+·", "page")),
}

var List = ListKeys{
	// Matched by the bubbles list itself; declared here so the footer can
	// advertise it from the same place as everything else.
	Navigate: key.NewBinding(key.WithHelp("↑/↓", "navigate")),
	Select:   key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "select")),
	New:      key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
	Delete:   key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
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
			bindings := make([]key.Binding, 0, 5)
			if !ctx.ListEmpty {
				bindings = append(bindings, List.Select)
			}
			bindings = append(bindings, List.New)
			if !ctx.ListEmpty {
				bindings = append(bindings, List.Delete)
			}

			return append(bindings, List.Navigate, Global.NextPanel)

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
			bindings := make([]key.Binding, 0, 3)
			if !ctx.ListEmpty {
				bindings = append(bindings, List.Select)
			}

			return append(bindings, List.Navigate, Global.NextPanel)

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
