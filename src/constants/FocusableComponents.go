package constants

// Component ids. A component compares the id it was built with against
// cmds.SetFocusMsg to decide whether it is focused, so these values are part of
// the focus protocol and must stay stable.
const (
	COMPONENT_MAIN_MENU    = 0
	COMPONENT_BODY_LIST    = 1
	COMPONENT_BODY_DETAILS = 2
)

// FocusableComponents are the component ids Tab cycles through, in order.
//
// The main menu is deliberately absent. Pages are switched with the global
// alt+<letter> chords that the nav underlines, so the nav never needs focus -
// which also removes the problem that it had no way to show whether it had
// focus. Tab therefore moves between the two body panels only, which is the
// movement that actually matters.
var FocusableComponents = []int{
	COMPONENT_BODY_LIST,
	COMPONENT_BODY_DETAILS,
}
