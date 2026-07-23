package cmds

import tea "charm.land/bubbletea/v2"

// OpenLogsModalMsg asks AppModel to open the streaming logs overlay for a
// single service (IsProfile false) or a whole profile (IsProfile true).
type OpenLogsModalMsg struct {
	Target    string
	IsProfile bool
}

func OpenLogsModal(target string, isProfile bool) tea.Cmd {
	return func() tea.Msg {
		return OpenLogsModalMsg{Target: target, IsProfile: isProfile}
	}
}

// LogLineMsg carries a single line read from the log stream.
type LogLineMsg string

// LogStreamEndedMsg is sent once the log channel closes (process exited or the
// stream was cancelled).
type LogStreamEndedMsg struct{}

// WaitForLog blocks on the next line from the stream channel and turns it into
// a message. The LogsModal re-issues this cmd after every LogLineMsg to pull
// the following line, which is how the stream keeps flowing through Bubble
// Tea's Update loop without a blocking read on the main goroutine.
func WaitForLog(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return LogStreamEndedMsg{}
		}
		return LogLineMsg(line)
	}
}
