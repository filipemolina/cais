package cmds

import (
	"github.com/filipemolina/stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

// ComposeFileListMsg carries the YAML files found in a directory, so the
// file picker modal can list them. Dir is the directory that was scanned,
// which the picker needs to turn a chosen name back into a path.
type ComposeFileListMsg struct {
	Dir   string
	Files []string
	Err   error
}

// ListComposeFiles scans dir for YAML files. The Files page's picker opens
// off the back of this, so the scan runs as a command rather than inline in
// AppModel's update.
func ListComposeFiles(dir string) tea.Cmd {
	return func() tea.Msg {
		files, err := utils.ListComposeFiles(dir)
		return ComposeFileListMsg{Dir: dir, Files: files, Err: err}
	}
}
