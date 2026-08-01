package cmds

import (
	"github.com/filipemolina/stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

type GetConfigMsg = struct {
	FileName string
	// Files is every compose-file candidate that exists in the directory, in
	// Docker's priority order, so Files[0] is FileName. The rest are the
	// candidates that lost - the footer counts them and the help overlay
	// lists them. Empty in tests that construct the message by hand.
	Files      []string
	Project    *types.Project
	EnvPath    string // Resolved .env path (compose-go semantics)
	EnvLoaded  bool   // Whether .env was actually consumed by compose-go
	Err        error
}

// GetConfig resolves the compose file for source and loads it. It re-resolves
// on every reload rather than remembering the winner, so a file created (or
// removed) while the app is running is picked up.
func GetConfig(source utils.ComposeSource) tea.Cmd {
	return func() tea.Msg {
		fileName, candidates, err := utils.GetComposeFileName(source)
		if err != nil {
			return GetConfigMsg{Err: err}
		}

		project, envPath, envLoaded, err := utils.ReadConfigFileExt(fileName)
		if err != nil {
			return GetConfigMsg{Err: err}
		}

		return GetConfigMsg{
			FileName:  fileName,
			Files:     candidates,
			Project:   project,
			EnvPath:   envPath,
			EnvLoaded: envLoaded,
		}
	}
}
