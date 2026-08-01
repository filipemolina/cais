package cmds

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// SearchDebounceInterval is how long the query input waits for another
// keystroke before firing a search - long enough to stay far under the Hub
// abuse limit on a burst of typing, short enough to still feel live (D3).
const SearchDebounceInterval = 350 * time.Millisecond

// SearchDebounceMsg fires after SearchDebounceInterval. Generation is
// stamped at the moment the timer was armed (the last keystroke), and the
// receiving component must compare it against its own current generation
// before firing a search - an older, superseded timer firing late must do
// nothing (D3).
type SearchDebounceMsg struct{ Generation int }

func SearchDebounce(generation int) tea.Cmd {
	return tea.Tick(SearchDebounceInterval, func(time.Time) tea.Msg {
		return SearchDebounceMsg{Generation: generation}
	})
}
