package cmds

import tea "charm.land/bubbletea/v2"

// SetComposeFileMsg carries the compose file the app resolved, so the footer
// can say which one won. Empty means none is loaded, which is the bootstrap
// state and the state left behind by a failed load.
//
// Docker resolves the file itself - every utils.DockerCompose* call shells out
// without -f - so this is a report, not a setting. Changing which file the app
// prefers would desync the panel from the commands it runs.
type SetComposeFileMsg string

func SetComposeFile(fileName string) tea.Cmd {
	return func() tea.Msg { return SetComposeFileMsg(fileName) }
}
