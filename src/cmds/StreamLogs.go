package cmds

import tea "charm.land/bubbletea/v2"

// OpenLogsModalMsg asks AppModel to open the streaming logs overlay for a
// single service (IsGroup false) or a whole group (IsGroup true).
type OpenLogsModalMsg struct {
	Target  string
	IsGroup bool
}

func OpenLogsModal(target string, isGroup bool) tea.Cmd {
	return func() tea.Msg {
		return OpenLogsModalMsg{Target: target, IsGroup: isGroup}
	}
}

// LogLinesMsg carries a batch of one or more lines read from the log stream.
// WaitForLog drains every line already buffered on the channel into one
// batch instead of returning after the first, so the initial `--tail` replay
// (which the producer sends in a tight burst) reaches the LogsModal as a
// single Update instead of one render per line - the render-per-line version
// was visible as the log scrolling to the bottom right after the modal
// opened, instead of opening already pinned there.
type LogLinesMsg []string

// LogStreamEndedMsg is sent once the log channel closes (process exited or the
// stream was cancelled).
type LogStreamEndedMsg struct{}

// WaitForLog blocks on the next line from the stream channel, then drains any
// further lines already waiting without blocking, and turns them into one
// batch message. The LogsModal re-issues this cmd after every LogLinesMsg to
// pull the following batch, which is how the stream keeps flowing through
// Bubble Tea's Update loop without a blocking read on the main goroutine.
func WaitForLog(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return LogStreamEndedMsg{}
		}

		lines := []string{line}
		for {
			select {
			case line, ok := <-ch:
				if !ok {
					return LogLinesMsg(lines)
				}
				lines = append(lines, line)
			default:
				return LogLinesMsg(lines)
			}
		}
	}
}
