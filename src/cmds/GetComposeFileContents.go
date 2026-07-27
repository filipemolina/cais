package cmds

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

// ComposeFileContentsMsg carries the raw text of a compose file for the
// Files page's read-only viewport. Name is the path it was read from;
// Contents is the file's bytes as-is (comments, blank lines and all).
type ComposeFileContentsMsg struct {
	Name     string
	Contents string
	Err      error
}

// GetComposeFileContents reads the compose file at fileName and returns
// its raw text. Called when the Files page becomes active and after any
// write through the app, so the viewport stays in sync with disk.
func GetComposeFileContents(fileName string) tea.Cmd {
	return func() tea.Msg {
		raw, err := os.ReadFile(fileName)
		if err != nil {
			return ComposeFileContentsMsg{
				Name: fileName,
				Err:  fmt.Errorf("reading %s: %w", fileName, err),
			}
		}

		return ComposeFileContentsMsg{Name: fileName, Contents: string(raw)}
	}
}
