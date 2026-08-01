package cmds

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/filipemolina/stack-stitcher/src/utils"
)

type SaveEnvFileMsg struct {
	Path string // The .env file path
	Err  error
}

// SaveEnvFile applies edits to the .env file and returns a message.
func SaveEnvFile(envPath string, ops []utils.EnvEditOp) tea.Cmd {
	return func() tea.Msg {
		if err := utils.ApplyEnvEdit(envPath, ops); err != nil {
			return SaveEnvFileMsg{
				Path: envPath,
				Err:  fmt.Errorf("failed to save .env: %w", err),
			}
		}
		return SaveEnvFileMsg{Path: envPath}
	}
}
