package cmds

import tea "charm.land/bubbletea/v2"

// SetComposeFileMsg carries the compose file the app resolved, so the footer
// can say which one won - and how many others were in the running. Docker
// resolves the file itself (every utils.DockerCompose* call shells out without
// -f), so this is a report, not a setting.
type SetComposeFileMsg struct {
	// Name is the winning candidate. Empty means none is loaded, which is the
	// bootstrap state and the state left behind by a failed load.
	Name string
	// Others are the candidates that lost, in priority order. The footer
	// marks the winner with +N when there are any; the help overlay lists
	// them. Empty when the winner was the only candidate.
	Others []string
}

func SetComposeFile(name string, others []string) tea.Cmd {
	return func() tea.Msg { return SetComposeFileMsg{Name: name, Others: others} }
}
